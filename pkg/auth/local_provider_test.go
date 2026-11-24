package auth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/harvester/harvester/pkg/auth/pbkdf2"
	harvesterfake "github.com/harvester/harvester/pkg/generated/clientset/versioned/fake"
	"github.com/harvester/harvester/pkg/util/fakeclients"
	mgmtv3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage/names"
	k8sTesting "k8s.io/client-go/testing"
)

func Test_GetTokenAuthFromRequest(t *testing.T) {
	assert := require.New(t)

	var tests = []struct {
		name                    string
		expectedAuthHeaderValue string
		headerFunction          func(*http.Request)
	}{
		{
			name:                    "basic auth",
			expectedAuthHeaderValue: "user:pass",
			headerFunction: func(r *http.Request) {
				r.SetBasicAuth("user", "pass")
			},
		},
		{
			name:                    "bearer auth",
			expectedAuthHeaderValue: "user:pass",
			headerFunction: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+"user:pass")
			},
		},
		{
			name:                    "using R_SESS cookie",
			expectedAuthHeaderValue: "user:pass",
			headerFunction: func(r *http.Request) {
				cookie := &http.Cookie{Name: "R_SESS", Value: "user:pass"}
				r.AddCookie(cookie)
			},
		},
	}

	// verify the token auth is extracted correctly

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1-login", nil)
			tt.headerFunction(req)
			assert.Equal(GetTokenAuthFromRequest(req), tt.expectedAuthHeaderValue)
		})
	}
}

// Test_Login verifies login functionality and ensures password match and token generation is correct
func Test_Login(t *testing.T) {

	assert := require.New(t)
	logrus.SetLevel(logrus.DebugLevel)
	const userName = "user-pkhgx"
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: pbkdf2.LocalUserPasswordsNamespace,
		},
	}

	userObj := &mgmtv3.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: userName,
		},
		Username:    "admin",
		DisplayName: "Admin User",
		Description: "",
	}

	// attempt login to verify http handler and token creation
	l := clientWrapper([]runtime.Object{namespace, userObj})

	// setup user password
	err := l.CreatePassword(userObj, "password1234")
	assert.NoError(err)

	req, err := http.NewRequest(http.MethodPost, "/v1-public/login", bytes.NewReader([]byte(`{"configName":"local","type":"localProvider","description":"UI session","responseType":"cookie","username":"admin","password":"password1234"}`)))
	assert.NoError(err)

	rr := httptest.NewRecorder()
	l.setup().ServeHTTP(rr, req)
	// verify request was successful
	assert.Equal(http.StatusOK, rr.Code)

	// extract token from cookie and verify backend object
	var token, tokenValue string
	cookies := rr.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == CookieName {
			assert.NotEmpty(cookie.Value)
			token, tokenValue = GetTokenKeyValue(cookie.Value)
		}

	}
	assert.NotEmpty(token)
	assert.NotEmpty(tokenValue)

	// check token CRD exists
	tokenObj, err := l.TokenCache.Get(token)
	assert.NoError(err)
	assert.Equal(tokenValue, tokenObj.Token)

}

// clientWrapper sets up the fakeclient with a custom reactor for create actions
func clientWrapper(objects []runtime.Object) *localAuthProvider {
	client := harvesterfake.NewSimpleClientset(objects...)
	client.PrependReactor(
		"create", "*",
		func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
			ret = action.(k8sTesting.CreateAction).GetObject()
			meta, ok := ret.(metav1.Object)
			if !ok {
				return
			}

			if meta.GetName() == "" && meta.GetGenerateName() != "" {
				meta.SetName(names.SimpleNameGenerator.GenerateName(meta.GetGenerateName()))
			}

			return
		})

	secretClient := fakeclients.SecretClient(client.CoreV1().Secrets)
	secretCache := fakeclients.SecretCache(client.CoreV1().Secrets)
	tokenClient := fakeclients.TokenClient(client.ManagementV3().Tokens)
	tokenCache := fakeclients.TokenCache(client.ManagementV3().Tokens)
	userCache := fakeclients.UserCache(client.ManagementV3().Users)
	credManager := pbkdf2.New(secretCache, secretClient)

	// attempt login to verify http handler and token creation
	return &localAuthProvider{
		ctx:         context.TODO(),
		Pbkdf2:      credManager,
		TokenClient: tokenClient,
		TokenCache:  tokenCache,
		UserCache:   userCache,
	}

}

// Test_Logout verifies logout functionality and verifies token deletion
func Test_Logout(t *testing.T) {
	assert := require.New(t)
	logrus.SetLevel(logrus.DebugLevel)
	tokenObj := &mgmtv3.Token{
		ObjectMeta: metav1.ObjectMeta{
			Name: "token-1",
		},
		Token: "key",
	}
	l := clientWrapper([]runtime.Object{tokenObj})
	req, err := http.NewRequest(http.MethodPost, "/v1-public/logout", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "token-1:key"})
	assert.NoError(err)
	rr := httptest.NewRecorder()
	l.setup().ServeHTTP(rr, req)
	assert.Equal(http.StatusOK, rr.Code)
	_, err = l.TokenCache.Get("token-1")
	assert.True(apierrors.IsNotFound(err))
}

// Test_Verify checks token verification
func Test_Verify(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	assert := require.New(t)
	tokenObj := &mgmtv3.Token{
		ObjectMeta: metav1.ObjectMeta{
			Name: "token-1",
		},
		Token:     "key",
		ExpiresAt: metav1.Now().Add(TTLMillis * time.Millisecond).UTC().Format(time.RFC3339),
	}
	l := clientWrapper([]runtime.Object{tokenObj})
	req, err := http.NewRequest(http.MethodGet, "/v1", nil)
	assert.NoError(err)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "token-1:key"})
	rr := httptest.NewRecorder()
	l.VerifyToken(rr, req)
	t.Log(rr.Body.String())
	assert.Equal(http.StatusOK, rr.Code)
}
