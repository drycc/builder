package conf

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/drycc/builder/pkg/sys"
)

const (
	storageCredLocation = "/var/run/secrets/drycc/minio/creds/"
	minioEndpointVar    = "DRYCC_MINIO_ENDPOINT"
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
	params["builder-bucket"] = "builder" // default
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

		params[file.Name()] = string(data)
	}
	params["bucket"] = params["builder-bucket"]
	mEndpointVar := env.Get(minioEndpointVar)
	params["region"] = "us-east-1"
	params["regionendpoint"] = fmt.Sprintf("http://%s", mEndpointVar)
	params["secure"] = false
	return params, nil
}
