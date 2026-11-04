package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathConfigValidate(t *testing.T) {
	var validConfigs []WatchPathConfig
	validConfigs = append(validConfigs, WatchPathConfig{
		Directory:  testDirectory,
		Path:       testPath,
		PathRegexp: "",
	})
	validConfigs = append(validConfigs, WatchPathConfig{
		Directory:  testDirectory,
		Path:       "",
		PathRegexp: testPath,
	})
	for _, config := range validConfigs {
		err := config.Validate()
		assert.NoError(t, err)
	}

	var invalidConfigs []WatchPathConfig
	// empty dir
	invalidConfigs = append(invalidConfigs, WatchPathConfig{
		Directory:  "",
		Path:       "",
		PathRegexp: "",
	})
	// relative dir
	invalidConfigs = append(invalidConfigs, WatchPathConfig{
		Directory:  "foo",
		Path:       testPath,
		PathRegexp: "",
	})
	// both path and path regexp
	invalidConfigs = append(invalidConfigs, WatchPathConfig{
		Directory:  testDirectory,
		Path:       testPath,
		PathRegexp: testPath,
	})
	// absolute path
	invalidConfigs = append(invalidConfigs, WatchPathConfig{
		Directory:  testDirectory,
		Path:       "/bar",
		PathRegexp: "",
	})
	// invalid regexp
	invalidConfigs = append(invalidConfigs, WatchPathConfig{
		Directory:  testDirectory,
		Path:       "",
		PathRegexp: "][",
	})
	for _, config := range invalidConfigs {
		err := config.Validate()
		assert.Error(t, err)
	}
}
