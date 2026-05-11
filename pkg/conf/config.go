// Package conf provides configuration utilities for the builder service.
package conf

import (
	"net"
	"net/url"
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

// Parameters is map which contains storage params
type Parameters map[string]any

// GetStorageParams returns the credentials required for connecting to object storage
func GetStorageParams(env sys.Env) (Parameters, error) {
	params := make(map[string]any)

	mEndpoint := env.Get(storageEndpointEnvVar)
	params["regionendpoint"] = mEndpoint
	region := "us-east-1" // region is required in distribution
	if endpointURL, err := url.Parse(mEndpoint); err == nil {
		if endpointURL.Hostname() != "" && net.ParseIP(endpointURL.Hostname()) == nil {
			region = strings.Split(endpointURL.Hostname(), ".")[0]
		}
	}
	params["region"] = region

	params["accesskey"] = env.Get(storageAccesskeyEnvVar)
	params["secretkey"] = env.Get(storageSecretkeyEnvVar)
	params["bucket"] = env.Get(storageBucketEnvVar)
	if env.Get(storagePathStyleEnvVar) == "true" {
		params["forcepathstyle"] = "true"
	}
	return params, nil
}
