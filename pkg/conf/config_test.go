package conf

import (
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
		"DRYCC_STORAGE_PATH_STYLE": "true",
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
