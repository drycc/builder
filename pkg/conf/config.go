package conf

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/drycc/builder/pkg/sys"
)

const (
	storageBucketEnvVar    = "DRYCC_STORAGE_BUCKET"
	storageEndpointEnvVar  = "DRYCC_STORAGE_ENDPOINT"
	storageAccesskeyEnvVar = "DRYCC_STORAGE_ACCESSKEY"
	storageSecretkeyEnvVar = "DRYCC_STORAGE_SECRETKEY"
	storagePathStyleEnvVar = "DRYCC_STORAGE_PATH_STYLE"
)

// ServiceKeyLocation holds the path of the service key secret.
var ServiceKeyLocation = "/var/run/secrets/drycc/controller/service-key"

// Parameters is map which contains storage params
type Parameters map[string]interface{}

// GetServiceKey returns the key to be used as token to interact with drycc-controller
func GetServiceKey() (string, error) {
	serviceKeyBytes, err := os.ReadFile(ServiceKeyLocation)
	if err != nil {
		return "", fmt.Errorf("couldn't get builder key from %s (%s)", ServiceKeyLocation, err)
	}
	serviceKey := strings.Trim(string(serviceKeyBytes), "\n")
	return serviceKey, nil
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
	if env.Get(storagePathStyleEnvVar) == "on" {
		params["forcepathstyle"] = "true"
	}

	return params, nil
}
