package gitreceive

import (
	"errors"
	"fmt"

	"github.com/drycc/builder/pkg/sys"
)

var (
	requiredEnvNames = []string{
		"DRYCC_STORAGE_BUCKET",
		"DRYCC_STORAGE_ENDPOINT",
		"DRYCC_STORAGE_PATH_STYLE",
		"DRYCC_REGISTRY_HOST",
	}
)

func checkImagebuilderRequiredEnv(imagebuilderEnv map[string]string) error {
	for index := range requiredEnvNames {
		envName := requiredEnvNames[index]
		if _, hasKey := imagebuilderEnv[envName]; !hasKey {
			msg := fmt.Sprintf("the environment variable %s is required", envName)
			return errors.New(msg)
		}
	}
	if imagebuilderEnv["DRYCC_REGISTRY_LOCATION"] == "off-cluster" {
		if imagebuilderEnv["DRYCC_REGISTRY_ORGANIZATION"] == "" {
			return errors.New("the environment variable DRYCC_REGISTRY_ORGANIZATION is required")
		}
	} else {
		if imagebuilderEnv["DRYCC_REGISTRY_PROXY_HOST"] == "" {
			return errors.New("the environment variable DRYCC_REGISTRY_PROXY_HOST is required")
		}
	}
	return nil
}

func getImagebuilderEnv(image *string, config *Config, env sys.Env) (map[string]string, error) {
	imagebuilderEnv := env.Environ([]string{"DRYCC_REGISTRY_", "DRYCC_STORAGE_"})
	if err := checkImagebuilderRequiredEnv(imagebuilderEnv); err != nil {
		return nil, err
	}
	if imagebuilderEnv["DRYCC_REGISTRY_LOCATION"] == "off-cluster" {
		*image = fmt.Sprintf(
			"%s/%s/%s",
			imagebuilderEnv["DRYCC_REGISTRY_HOST"],
			imagebuilderEnv["DRYCC_REGISTRY_ORGANIZATION"],
			*image,
		)
	} else {
		imagebuilderEnv["DRYCC_REGISTRY_ORGANIZATION"] = config.App()
		*image = fmt.Sprintf(
			"%s/%s/%s",
			imagebuilderEnv["DRYCC_REGISTRY_PROXY_HOST"],
			imagebuilderEnv["DRYCC_REGISTRY_ORGANIZATION"],
			*image,
		)
	}
	return imagebuilderEnv, nil
}
