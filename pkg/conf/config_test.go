package conf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/drycc/builder/pkg/sys"
	"github.com/stretchr/testify/assert"
)

func TestGetStorageParams(t *testing.T) {
	env := sys.NewFakeEnv()
	env.Envs = map[string]string{
		"DRYCC_STORAGE_BUCKET":     "builder",
		"DRYCC_STORAGE_ENDPOINT":   "http://localhost:8088",
		"DRYCC_STORAGE_ACCESSKEY":  "admin",
		"DRYCC_STORAGE_SECRETKEY":  "adminpass",
		"DRYCC_STORAGE_PATH_STYLE": "on",
	}
	params, err := GetStorageParams(env)
	if err != nil {
		t.Errorf("received error while retrieving storage params: %v", err)
	}
	assert.Equal(t, params["forcepathstyle"], "true", "forcepathstyle")
	assert.Equal(t, params["regionendpoint"], "http://localhost:8088", "region endpoint")
	assert.Equal(t, params["region"], "localhost", "region")
	assert.Equal(t, params["bucket"], "builder", "bucket")
	assert.Equal(t, params["accesskey"], "admin", "accesskey")
	assert.Equal(t, params["secretkey"], "adminpass", "secretkey")
}

func TestGetControllerClient(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove service-key from %s (%s)", tmpDir, err)
		}
	}()

	ServiceKeyLocation = filepath.Join(tmpDir, "service-key")
	data := []byte("testbuilderkey")
	if err := os.WriteFile(ServiceKeyLocation, data, 0o644); err != nil {
		t.Fatalf("error creating %s (%s)", ServiceKeyLocation, err)
	}

	key, err := GetServiceKey()
	assert.Equal(t, err, nil)
	assert.Equal(t, key, string(data), "data")
}

func TestGetServiceKeyError(t *testing.T) {
	_, err := GetServiceKey()
	assert.True(t, err != nil, "no error received when there should have been")
}
