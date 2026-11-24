package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/harvester/harvester/pkg/auth/pbkdf2"
	"github.com/harvester/harvester/pkg/config"
	"github.com/harvester/harvester/pkg/util"
	"github.com/rancher/apiserver/pkg/apierror"
	mgmtv3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	rancherv3 "github.com/rancher/rancher/pkg/generated/controllers/management.cattle.io/v3"
	"github.com/rancher/wrangler/v3/pkg/randomtoken"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/authentication/user"
)

const (
	Name            = "local"
	TTLMillis       = 57600000 //16 * time.Hour * time.Second * time.Millisecond
	ResponseType    = "cookie"
	Description     = "UI Session"
	CookieName      = "R_SESS"
	UserIDLabel     = "authn.management.cattle.io/token-userId"
	TokenKindLabel  = "authn.management.cattle.io/kind"
	AuthHeaderName  = "Authorization"
	AuthValuePrefix = "Bearer"
	BasicAuthPrefix = "Basic"
)

// implementation of the local authentication provider
type LocalAuthProvider struct {
	ctx context.Context
	*pbkdf2.Pbkdf2
	TokenCache  rancherv3.TokenCache
	TokenClient rancherv3.TokenClient
	UserCache   rancherv3.UserCache
}

func newLocalProvider(ctx context.Context, scaled *config.Management) *LocalAuthProvider {
	return &LocalAuthProvider{
		ctx:         ctx,
		Pbkdf2:      pbkdf2.New(scaled.CoreFactory.Core().V1().Secret().Cache(), scaled.CoreFactory.Core().V1().Secret()),
		TokenCache:  scaled.RancherManagementFactory.Management().V3().Token().Cache(),
		TokenClient: scaled.RancherManagementFactory.Management().V3().Token(),
	}
}

// Login validates the password and generates a token associated with the user session
func (l *LocalAuthProvider) Login(w http.ResponseWriter, r *http.Request) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("login: Error reading request body: %s", err)
		util.ReturnAPIError(w, apierror.NewAPIError(validation.InvalidBodyContent, ""))
		return
	}
	loginObj := &mgmtv3.BasicLogin{}
	if err := json.Unmarshal(bytes, loginObj); err != nil {
		logrus.Errorf("login: Error parsing request body: %s", err)
		util.ReturnAPIError(w, apierror.NewAPIError(validation.InvalidBodyContent, ""))
		return
	}

	// looking up userObj related to userName in loginObj
	userObjs, err := l.UserCache.List(labels.Everything())
	if err != nil {
		util.ReturnAPIError(w, apierror.NewAPIError(validation.Unauthorized, ""))
		return
	}

	if len(userObjs) == 0 {
		util.ReturnAPIError(w, apierror.NewAPIError(validation.Unauthorized, "no local users found"))
		return
	}

	matchingUserObjs := []*mgmtv3.User{}
	for _, u := range userObjs {
		if u.Username == loginObj.Username {
			matchingUserObjs = append(matchingUserObjs, u)
		}
	}

	if len(matchingUserObjs) > 1 {
		logrus.Warnf("Multiple users found for username %s", loginObj.Username)
		util.ReturnAPIError(w, apierror.NewAPIError(validation.Unauthorized, "found more than one user for username"))
		return
	}

	userObj := matchingUserObjs[0]

	err = l.VerifyPassword(userObj, loginObj.Password)
	if err != nil {
		logrus.Error(err)
		util.ReturnAPIError(w, apierror.NewAPIError(validation.Unauthorized, ""))
		return
	}

	logrus.Debugf("Password verified for user %s", userObj.Username)
	// extract user info and create token
	principalID := getLocalPrincipalID(userObj)
	userPrincipal := toPrincipal("user", userObj.DisplayName, userObj.Username, principalID)
	userPrincipal.Me = true
	groupPrincipals := []mgmtv3.Principal{}

	// login for generating token here...
	// TODO: get groupPrincipals, providerToken, ttl, description
	// TODO: also identify kubeconfig token generation logic.. or we just move it download rke2 kubeconfig

	tokenObj := &mgmtv3.Token{
		UserPrincipal:   userPrincipal,
		GroupPrincipals: groupPrincipals,
		IsDerived:       false,
		ExpiresAt:       metav1.Now().Add(TTLMillis * time.Millisecond).UTC().Format(time.RFC3339),
		TTLMillis:       TTLMillis,
		UserID:          userObj.Name,
		AuthProvider:    Name, // local provider name
		Description:     Description,
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				TokenKindLabel: "session",
			},
		},
	}
	token, tokenKey, err := l.createToken(tokenObj)

	if err != nil {
		logrus.Errorf("login: Error creating login token for user %s: %v", userObj.Name, err)
		util.ReturnAPIError(w, apierror.NewAPIError(validation.ServerError, ""))
		return
	}

	bearerToken := token.Name + ":" + tokenKey

	tokenCookie := &http.Cookie{
		Name:     CookieName,
		Value:    bearerToken,
		Secure:   true,
		Path:     "/",
		HttpOnly: true,
		// Lax is the default in most browsers; setting it
		// explicitly is a good security measure.
		SameSite: http.SameSiteLaxMode,
	}
	logrus.Debugf("User %s authenticated and logged in successfully", userObj.Name)
	http.SetCookie(w, tokenCookie)
	w.WriteHeader(http.StatusOK)
}

// Logout uses the token header to verify session and perform logout by deleting the token object
func (l *LocalAuthProvider) Logout(w http.ResponseWriter, r *http.Request) {
	tokenHeader := GetTokenAuthFromRequest(r)
	tokenName, _ := GetTokenKeyValue(tokenHeader)

	err := l.TokenClient.Delete(tokenName, &metav1.DeleteOptions{})
	if err != nil {
		// A `token not found` is intentionally treated as success.
		// Not doing so will cause multiple concurrent deletes of the
		// same token to throw an error (except the first) in a
		// fundamentally benign situation.
		//
		// Regarding the overall issue fixed here this behaviour was
		// only contributory, by making the main issue invisible, which
		// was the misrouting of tokens by the caller. Adding logging is
		// enough to address that.
		if apierrors.IsNotFound(err) {
			logrus.Debugf("token to delete (%s) not found", tokenName)
			return
		}
		return
	}
	logrus.Debug("Deleted Token")
}

// Update password allows users to update their local password
func (l *LocalAuthProvider) UpdatePassword(w http.ResponseWriter, r *http.Request) {

}

func (l *LocalAuthProvider) VerifyToken(w http.ResponseWriter, r *http.Request, next http.Handler) {
	tokenHeader := GetTokenAuthFromRequest(r)
	tokenName, tokenValue := GetTokenKeyValue(tokenHeader)
	// no tokenName or tokenValue present, request is unauthorised so we redirect back to login page
	if tokenValue == "" || tokenName == "" {
		logrus.Error("missing token")
		http.Redirect(w, r, "/v1-public/login", http.StatusFound)
		return
	}
	// fetch tokenName from cache to verify
	tokenObj, err := l.TokenClient.Get(tokenName, metav1.GetOptions{})
	if err != nil {
		logrus.Error("invalid token")
		http.Redirect(w, r, "/v1-public/login", http.StatusFound)
		return
	}
	if tokenObj.Token != tokenValue {
		logrus.Error("invalid token")
		http.Redirect(w, r, "/v1-public/login", http.StatusFound)
		return
	}
	// check if token is expired
	if tokenObj.ExpiresAt != "" {
		tokenExpiryTime, err := time.Parse(time.RFC3339, tokenObj.ExpiresAt)
		if err != nil {
			http.Error(w, "unable to parse token expiry", http.StatusInternalServerError)
			return
		}
		if tokenExpiryTime.Before(time.Now()) {
			logrus.Error("token expired")
			http.Redirect(w, r, "/v1-public/login", http.StatusFound)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	next.ServeHTTP(w, r)
}

func (l *LocalAuthProvider) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.VerifyToken(w, r, next)
	})
}

// LoginHandler sets up the wiring for the various local login handler endpoints
func LoginHandler(ctx context.Context, scaled *config.Management) *LocalAuthProvider {
	return newLocalProvider(ctx, scaled)

}

// setup is a wrapper function defined to make unit tests easier
func (l *LocalAuthProvider) Routes() http.Handler {
	r := http.NewServeMux()
	r.HandleFunc("POST /v1-public/login", l.Login)
	r.HandleFunc("POST /v1-public/logout", l.Logout)
	return r
}

func getLocalPrincipalID(user *mgmtv3.User) string {
	// TODO error condition handling: no principal, more than one that would match
	var principalID string
	for _, p := range user.PrincipalIDs {
		if strings.HasPrefix(p, Name+"://") {
			principalID = p
		}
	}
	if principalID == "" {
		return Name + "://" + user.Name
	}
	return principalID
}

func toPrincipal(principalType, displayName, loginName, id string) mgmtv3.Principal {
	if displayName == "" {
		displayName = loginName
	}

	princ := mgmtv3.Principal{
		ObjectMeta:    metav1.ObjectMeta{Name: id},
		DisplayName:   displayName,
		LoginName:     loginName,
		Provider:      Name,
		PrincipalType: principalType,
	}

	return princ
}

// createToken returns the token object and it's unhashed token key, which is stored hashed
func (l *LocalAuthProvider) createToken(k8sToken *mgmtv3.Token) (*mgmtv3.Token, string, error) {
	key, err := randomtoken.Generate()
	if err != nil {
		logrus.Errorf("Failed to generate token key: %v", err)
		return nil, "", errors.New("failed to generate token key")
	}

	if k8sToken.ObjectMeta.Labels == nil {
		k8sToken.ObjectMeta.Labels = make(map[string]string)
	}
	k8sToken.APIVersion = "management.cattle.io/v3"
	k8sToken.Kind = "Token"
	k8sToken.Token = key
	k8sToken.ObjectMeta.Labels[UserIDLabel] = k8sToken.UserID
	k8sToken.ObjectMeta.GenerateName = "token-"
	createdToken, err := l.TokenClient.Create(k8sToken)

	if err != nil {
		return nil, "", err
	}

	return createdToken, key, nil
}

// extract token from request
func GetTokenAuthFromRequest(req *http.Request) string {
	var tokenAuthValue string
	authHeader := req.Header.Get(AuthHeaderName)
	authHeader = strings.TrimSpace(authHeader)

	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if strings.EqualFold(parts[0], AuthValuePrefix) {
			if len(parts) > 1 {
				tokenAuthValue = strings.TrimSpace(parts[1])
			}
		} else if strings.EqualFold(parts[0], BasicAuthPrefix) {
			if len(parts) > 1 {
				base64Value := strings.TrimSpace(parts[1])
				data, err := base64.URLEncoding.DecodeString(base64Value)
				if err != nil {
					logrus.Errorf("Error %v parsing %v header", err, AuthHeaderName)
				} else {
					tokenAuthValue = string(data)
				}
			}
		}
	} else {
		cookie, err := req.Cookie(CookieName)
		if err == nil {
			tokenAuthValue = cookie.Value
		}
	}

	return tokenAuthValue
}

func GetTokenKeyValue(tokenHeader string) (string, string) {
	parts := strings.SplitN(tokenHeader, ":", 2)
	return parts[0], parts[1]
}

// verifies token is valid and matches a user and defaults to admin user
func (l *LocalAuthProvider) Authenticate(r *http.Request) (user.Info, bool, error) {
	tokenHeader := GetTokenAuthFromRequest(r)
	tokenName, tokenValue := GetTokenKeyValue(tokenHeader)
	user := &user.DefaultInfo{}
	// no tokenName or tokenValue present, request is unauthorised so we redirect back to login page
	if tokenValue == "" || tokenName == "" {
		return user, false, fmt.Errorf("missing token or value")
	}
	// fetch tokenName from cache to verify
	tokenObj, err := l.TokenClient.Get(tokenName, metav1.GetOptions{})
	if err != nil {
		return user, false, fmt.Errorf("token not found")
	}
	if tokenObj.Token != tokenValue {
		return user, false, fmt.Errorf("invalid token")
	}
	// check if token is expired
	if tokenObj.ExpiresAt != "" {
		tokenExpiryTime, err := time.Parse(time.RFC3339, tokenObj.ExpiresAt)
		if err != nil {
			return user, false, fmt.Errorf("invalid expiry")
		}
		if tokenExpiryTime.Before(time.Now()) {
			return user, false, fmt.Errorf("token expired")
		}
	}

	// find user reference from token
	user.Name = "admin"
	user.UID = "admin"
	user.Groups = []string{
		"system:masters",
		"system:authenticated",
	}
	return user, true, nil
}
