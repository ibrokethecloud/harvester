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
	commonObjs := []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: pbkdf2.LocalUserPasswordsNamespace,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userName,
				Namespace: pbkdf2.LocalUserPasswordsNamespace,
				Annotations: map[string]string{
					"cattle.io/password-hash": "pbkdf2sha3512",
				},
			},
			Data: map[string][]byte{
				"password": []byte("gthSMwMH6Q/JVnpaTk9naITMuQMMTfb7EEOXKdzueXg="),
				"salt":     []byte("hvYsPN+w+kgM+Jf+lqgcW3yWslcd/aSjxR0kZg5Wjf8="),
			},
		},
		&mgmtv3.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user-pkhgx",
			},
			Username:    "admin",
			DisplayName: "Admin User",
			Description: "",
		},
	}

	client := harvesterfake.NewSimpleClientset(commonObjs...)
	secretClient := fakeclients.SecretClient(client.CoreV1().Secrets)
	secretCache := fakeclients.SecretCache(client.CoreV1().Secrets)
	tokenClient := fakeclients.TokenClient(client.ManagementV3().Tokens)
	tokenCache := fakeclients.TokenCache(client.ManagementV3().Tokens)
	userCache := fakeclients.UserCache(client.ManagementV3().Users)
	l := &localAuthProvider{
		ctx:         context.TODO(),
		Pbkdf2:      pbkdf2.New(secretCache, secretClient),
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
