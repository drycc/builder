package conf

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/arschles/assert"
	"github.com/drycc/builder/pkg/sys"
)

func TestGetStorageParams(t *testing.T) {

	env := sys.NewFakeEnv()
	env.Envs = map[string]string{
		"BUILDER_STORAGE":       "minio",
		"DRYCC_MINIO_LOOKUP":    "path",
		"DRYCC_MINIO_BUCKET":    "builder",
		"DRYCC_MINIO_ENDPOINT":  "http://localhost:8088",
		"DRYCC_MINIO_ACCESSKEY": "admin",
		"DRYCC_MINIO_SECRETKEY": "adminpass",
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
	tmpDir, err := ioutil.TempDir("", "tmpdir")
	if err != nil {
		t.Fatalf("error creating temp directory (%s)", err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("failed to remove builder-key from %s (%s)", tmpDir, err)
		}
	}()

	BuilderKeyLocation = filepath.Join(tmpDir, "builder-key")
	data := []byte("testbuilderkey")
	if err := ioutil.WriteFile(BuilderKeyLocation, data, 0644); err != nil {
		t.Fatalf("error creating %s (%s)", BuilderKeyLocation, err)
	}

	key, err := GetBuilderKey()
	assert.NoErr(t, err)
	assert.Equal(t, key, string(data), "data")
}

func TestGetBuilderKeyError(t *testing.T) {
	_, err := GetBuilderKey()
	assert.True(t, err != nil, "no error received when there should have been")
}
