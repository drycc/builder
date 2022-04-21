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
	minioLookupEnvVar    = "DRYCC_MINIO_LOOKUP"
	minioBucketEnvVar    = "DRYCC_MINIO_BUCKET"
	minioEndpointEnvVar  = "DRYCC_MINIO_ENDPOINT"
	minioAccesskeyEnvVar = "DRYCC_MINIO_ACCESSKEY"
	minioSecretkeyEnvVar = "DRYCC_MINIO_SECRETKEY"
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

	mEndpoint := env.Get(minioEndpointEnvVar)
	params["regionendpoint"] = mEndpoint
	region := "us-east-1" //region is required in distribution
	if endpointURL, err := url.Parse(mEndpoint); err == nil {
		if endpointURL.Hostname() != "" && net.ParseIP(endpointURL.Hostname()) == nil {
			region = strings.Split(endpointURL.Hostname(), ".")[0]
		}
	}
	params["region"] = region

	params["accesskey"] = env.Get(minioAccesskeyEnvVar)
	params["secretkey"] = env.Get(minioSecretkeyEnvVar)
	params["bucket"] = env.Get(minioBucketEnvVar)
	if env.Get(minioLookupEnvVar) == "path" {
		params["forcepathstyle"] = "true"
	}

	return params, nil
}
