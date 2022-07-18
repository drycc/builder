package conf

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"strings"

	"github.com/drycc/builder/pkg/sys"
)

const (
	storageLookupEnvVar    = "DRYCC_STORAGE_LOOKUP"
	storageBucketEnvVar    = "DRYCC_STORAGE_BUCKET"
	storageEndpointEnvVar  = "DRYCC_STORAGE_ENDPOINT"
	storageAccesskeyEnvVar = "DRYCC_STORAGE_ACCESSKEY"
	storageSecretkeyEnvVar = "DRYCC_STORAGE_SECRETKEY"
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

	mEndpoint := env.Get(storageEndpointEnvVar)
	params["regionendpoint"] = mEndpoint
	region := "us-east-1" //region is required in distribution
	if endpointURL, err := url.Parse(mEndpoint); err == nil {
		if endpointURL.Hostname() != "" && net.ParseIP(endpointURL.Hostname()) == nil {
			region = strings.Split(endpointURL.Hostname(), ".")[0]
		}
	}
	params["region"] = region

	params["accesskey"] = env.Get(storageAccesskeyEnvVar)
	params["secretkey"] = env.Get(storageSecretkeyEnvVar)
	params["bucket"] = env.Get(storageBucketEnvVar)
	if env.Get(storageLookupEnvVar) == "path" {
		params["forcepathstyle"] = "true"
	}

	return params, nil
}
