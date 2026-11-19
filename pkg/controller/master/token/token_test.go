package token

import (
	"testing"

	fakegenerated "github.com/harvester/harvester/pkg/generated/clientset/versioned/fake"
	"github.com/harvester/harvester/pkg/util/fakeclients"
	mgmtv3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var (
	tokenYaml = `apiVersion: management.cattle.io/v3
authProvider: local
current: false
description: UI session
expired: false
expiresAt: "2026-11-19T16:09:43Z"
isDerived: false
kind: Token
lastUsedAt: "2026-11-19T00:09:50Z"
metadata:
  creationTimestamp: "2026-11-19T00:09:43Z"
  generateName: token-
  generation: 10
  labels:
    authn.management.cattle.io/kind: session
    authn.management.cattle.io/token-userId: user-b2jz9
  name: token-n7s7s
  resourceVersion: "310296023"
  uid: 2f6cc178-6f63-4793-a621-7449fd1dc035
token: "something"
ttl: 57600000
userId: user-b2jz9
userPrincipal:
  displayName: Default Admin
  loginName: admin
  me: true
  metadata:
    name: local://user-b2jz9
  principalType: user
  provider: local`
)

func Test_TokenCleanup(t *testing.T) {
	tests := []struct {
		name        string
		token       *mgmtv3.Token
		expectError bool
	}{
		{
			name: "token ignored because of deletion timestamp",
			token: &mgmtv3.Token{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "deletion-test",
					DeletionTimestamp: ptr.To(metav1.Now()),
				},
				ExpiresAt: "2026-11-19T16:09:43Z",
			},
			expectError: false,
		},
		{
			name: "token ignored because expiresAt is not set",
			token: &mgmtv3.Token{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-expiry-test",
				},
				ExpiresAt: "",
			},
			expectError: false,
		},
		{
			name: "token deleted due to expiry",
			token: &mgmtv3.Token{
				ObjectMeta: metav1.ObjectMeta{
					Name: "expired-token",
				},
				ExpiresAt: "2025-11-19T16:09:43Z",
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := require.New(t)
			harvFakeClient := fakegenerated.NewSimpleClientset()
			assert.NoError(harvFakeClient.Tracker().Add(tc.token), "expected no error while adding token to fake harvester client")
			fakeClient := fakeclients.TokenClient(harvFakeClient.ManagementV3().Tokens)
			handler := &TokenCleanupHandler{
				tokenClient: fakeClient,
			}
			_, err := handler.Cleanup(tc.token.Name, tc.token)
			assert.NoError(err, "expected no error during cleanup reconcile")

			// check if object is gone
			_, getError := fakeClient.Get(tc.token.Name, metav1.GetOptions{})
			if tc.expectError {
				// we expect to get object not found error
				assert.True(apierrors.IsNotFound(getError), "expected to get api object not found error")
			} else {
				assert.Nil(getError, "expected no error during reconcile, indicates token has not been GC'd")
			}
		})
	}
}
