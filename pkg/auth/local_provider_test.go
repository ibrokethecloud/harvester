package auth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harvester/harvester/pkg/auth/pbkdf2"
	harvesterfake "github.com/harvester/harvester/pkg/generated/clientset/versioned/fake"
	"github.com/harvester/harvester/pkg/util/fakeclients"
	mgmtv3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

	client := harvesterfake.NewSimpleClientset(namespace, userObj)
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
		},
	)
	secretClient := fakeclients.SecretClient(client.CoreV1().Secrets)
	secretCache := fakeclients.SecretCache(client.CoreV1().Secrets)
	tokenClient := fakeclients.TokenClient(client.ManagementV3().Tokens)
	tokenCache := fakeclients.TokenCache(client.ManagementV3().Tokens)
	userCache := fakeclients.UserCache(client.ManagementV3().Users)
	credManager := pbkdf2.New(secretCache, secretClient)

	// setup user password
	err := credManager.CreatePassword(userObj, "password1234")
	assert.NoError(err)

	// attempt login to verify http handler and token creation
	l := &localAuthProvider{
		ctx:         context.TODO(),
		Pbkdf2:      credManager,
		TokenClient: tokenClient,
		TokenCache:  tokenCache,
		UserCache:   userCache,
	}

	req, err := http.NewRequest(http.MethodPost, "/v1-public/login", bytes.NewReader([]byte(`{"configName":"local","type":"localProvider","description":"UI session","responseType":"cookie","username":"admin","password":"password1234"}`)))
	assert.NoError(err)

	rr := httptest.NewRecorder()
	l.setup().ServeHTTP(rr, req)
	// verify request was successful
	assert.Equal(http.StatusOK, rr.Code)

	t.Log(rr.Header())

}
