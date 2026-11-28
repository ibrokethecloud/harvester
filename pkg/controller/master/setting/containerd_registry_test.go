package setting

import (
	"testing"

	harvesterv1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	"github.com/stretchr/testify/require"
)

const (
	settingValue = `{
  "Mirrors": {
    "docker.io": {
      "Endpoints": ["https://myregistry.local:5000"],
      "Rewrites": null
    }
  },
  "Configs": {
    "myregistry.local:5000": {
      "Auth": {
        "Username": "testuser",
        "Password": "testpassword"
      },
      "TLS": {
        "InsecureSkipVerify": false
      }
    }
  }
}`
)

func Test_ContainerdRegistryWithCustomValue(t *testing.T) {
	setting := &harvesterv1.Setting{}
	setting.Value = settingValue
	assert := require.New(t)
	registry, err := generateRegistry(setting)
	assert.NoError(err)
	assert.Len(registry.Mirrors, 2, "expected to find 2 mirrors")
}

func Test_ContainerdRegistryWithEmptyValue(t *testing.T) {
	setting := &harvesterv1.Setting{}
	assert := require.New(t)
	registry, err := generateRegistry(setting)
	assert.NoError(err)
	assert.Len(registry.Mirrors, 1, "expected to find 1 mirrors")
}
