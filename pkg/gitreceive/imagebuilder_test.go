package gitreceive

import (
	"errors"
	"testing"

	"github.com/drycc/builder/pkg/sys"
	"github.com/stretchr/testify/assert"
)

func TestGetImagebuilderEnvOffclusterErr(t *testing.T) {
	image := "test-image"
	config := &Config{
		Repository: "python-getting-started.git",
	}
	env := sys.NewFakeEnv()
	env.Envs = map[string]string{
		"DRYCC_STORAGE_BUCKET":     "builder",
		"DRYCC_STORAGE_ENDPOINT":   "drycc-storage",
		"DRYCC_STORAGE_PATH_STYLE": "auto",
		"DRYCC_REGISTRY_LOCATION":  "off-cluster",
	}
	_, err := getImagebuilderEnv(&image, config, env)
	assert.Error(t, err, errors.New("the environment variable DRYCC_REGISTRY_HOST is required"))
	env.Envs["DRYCC_REGISTRY_HOST"] = "drycc-registry"
	_, err = getImagebuilderEnv(&image, config, env)
	assert.Error(t, err, errors.New("the environment variable DRYCC_REGISTRY_ORGANIZATION is required"))
}

func TestGetImagebuilderEnvOffclusterSuccess(t *testing.T) {
	env := sys.NewFakeEnv()
	env.Envs = map[string]string{
		"DRYCC_STORAGE_BUCKET":        "builder",
		"DRYCC_STORAGE_ENDPOINT":      "drycc-storage",
		"DRYCC_STORAGE_PATH_STYLE":    "auto",
		"DRYCC_REGISTRY_HOST":         "quay.io",
		"DRYCC_REGISTRY_ORGANIZATION": "kmala",
		"DRYCC_REGISTRY_LOCATION":     "off-cluster",
	}
	expectedData := map[string]string{
		"DRYCC_STORAGE_BUCKET":        "builder",
		"DRYCC_STORAGE_ENDPOINT":      "drycc-storage",
		"DRYCC_STORAGE_PATH_STYLE":    "auto",
		"DRYCC_REGISTRY_LOCATION":     "off-cluster",
		"DRYCC_REGISTRY_HOST":         "quay.io",
		"DRYCC_REGISTRY_ORGANIZATION": "kmala",
	}
	config := &Config{
		Repository: "python-getting-started.git",
	}
	expectedImage := "quay.io/kmala/test-image"

	image := "test-image"
	imagebuilderEnv, err := getImagebuilderEnv(&image, config, env)
	assert.Equal(t, err, nil)
	assert.Equal(t, expectedData, imagebuilderEnv, "registry details")

	assert.Equal(t, expectedImage, image, "image")
}

func TestGetImagebuilderEnvOnclusterSuccess(t *testing.T) {
	env := sys.NewFakeEnv()
	env.Envs = map[string]string{
		"DRYCC_STORAGE_BUCKET":      "builder",
		"DRYCC_STORAGE_ENDPOINT":    "drycc-storage",
		"DRYCC_STORAGE_PATH_STYLE":  "auto",
		"DRYCC_REGISTRY_HOST":       "drycc-registry",
		"DRYCC_REGISTRY_PROXY_HOST": "127.0.0.1:8000",
		"DRYCC_REGISTRY_LOCATION":   "on-cluster",
	}
	expectedData := map[string]string{
		"DRYCC_STORAGE_BUCKET":        "builder",
		"DRYCC_STORAGE_ENDPOINT":      "drycc-storage",
		"DRYCC_STORAGE_PATH_STYLE":    "auto",
		"DRYCC_REGISTRY_HOST":         "drycc-registry",
		"DRYCC_REGISTRY_LOCATION":     "on-cluster",
		"DRYCC_REGISTRY_PROXY_HOST":   "127.0.0.1:8000",
		"DRYCC_REGISTRY_ORGANIZATION": "python-getting-started",
	}
	config := &Config{
		Repository: "python-getting-started.git",
	}
	expectedImage := "127.0.0.1:8000/python-getting-started/python-getting-started-web:v1.2.1"

	image := "python-getting-started-web:v1.2.1"
	imagebuilderEnv, err := getImagebuilderEnv(&image, config, env)
	assert.Equal(t, err, nil)
	assert.Equal(t, expectedData, imagebuilderEnv, "registry details")

	assert.Equal(t, expectedImage, image, "image")
}
