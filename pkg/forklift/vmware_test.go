package forklift

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

var (
	credentials = &corev1.Secret{
		StringData: map[string]string{
			"insecureSkipVerify": "true",
		},
	}
)

func Test_GetDatastores(t *testing.T) {
	updateSecret()
	assert := require.New(t)
	provider, err := NewVsphereProvider(context.TODO(), credentials, "test-provider", "test-provider-id")
	assert.NoError(err)
	datastores, err := provider.GetDatastores()
	assert.NoError(err)
	t.Log(datastores)
	assert.NoError(provider.Close())
}

func Test_GetNetworks(t *testing.T) {
	updateSecret()
	assert := require.New(t)
	provider, err := NewVsphereProvider(context.TODO(), credentials, "test-provider", "test-provider-id")
	assert.NoError(err)
	networks, err := provider.GetNetworks()
	assert.NoError(err)
	t.Log(networks)
	assert.NoError(provider.Close())
}

func Test_GetVirtualMachines(t *testing.T) {
	updateSecret()
	assert := require.New(t)
	provider, err := NewVsphereProvider(context.TODO(), credentials, "test-provider", "test-provider-id")
	assert.NoError(err)
	virtualMachines, err := provider.GetVirtualMachines()
	assert.NoError(err)
	t.Log(virtualMachines)
	assert.NoError(provider.Close())
}

func updateSecret() {
	userName := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	url := os.Getenv("URL")

	credentials.StringData["user"] = userName
	credentials.StringData["password"] = password
	credentials.StringData["url"] = url
}
