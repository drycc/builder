package conf

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/drycc/builder/pkg/sys"
)

const (
	storageCredLocation = "/var/run/secrets/drycc/objectstore/creds/"
	minioHostEnvVar     = "DRYCC_MINIO_ENDPOINT"
	gcsKey              = "key.json"
)

// BuilderKeyLocation holds the path of the builder key secret.
var BuilderKeyLocation = "/var/run/secrets/api/auth/builder-key"

// Parameters is map which contains storage params
type Parameters map[string]interface{}

// GetBuilderKey returns the key to be used as token to interact with drycc-controller
func GetBuilderKey() (string, error) {
	builderKeyBytes, err := ioutil.ReadFile(BuilderKeyLocation)
	if err != nil {
		return "", fmt.Errorf("couldn't get builder key from %s (%s)", BuilderKeyLocation, err)
	}
	builderKey := strings.Trim(string(builderKeyBytes), "\n")
	return builderKey, nil
}

// GetStorageParams returns the credentials required for connecting to object storage
func GetStorageParams(env sys.Env) (Parameters, error) {
	params := make(map[string]interface{})
	files, err := ioutil.ReadDir(storageCredLocation)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() || file.Name() == "..data" {
			continue
		}
		data, err := ioutil.ReadFile(storageCredLocation + file.Name())
		if err != nil {
			return nil, err
		}
		//GCS expect the to have the location of the service account credential json file
		if file.Name() == gcsKey {
			params["keyfile"] = storageCredLocation + file.Name()
		} else {
			params[file.Name()] = string(data)
		}
	}
	params["bucket"] = params["builder-bucket"]
	mHost := env.Get(minioHostEnvVar)
	mPort := env.Get(minioPortEnvVar)
	params["region"] = "us-east-1"
	params["regionendpoint"] = fmt.Sprintf("http://%s:%s", mHost, mPort)
	params["secure"] = false
	return params, nil
}
