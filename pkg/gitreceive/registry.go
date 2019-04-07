package gitreceive

import (
	"strings"

	client "k8s.io/kubernetes/pkg/client/unversioned"
)

const (
	registrySecret = "registry-secret"
)

func getDetailsFromRegistrySecret(secretGetter client.SecretsInterface, secret string) (map[string]string, error) {
	regSecret, err := secretGetter.Get(secret)
	if err != nil {
		return nil, err
	}
	regDetails := make(map[string]string)
	for key, value := range regSecret.Data {
		regDetails[key] = string(value)
	}
	return regDetails, nil
}

func getRegistryDetails(kubeClient client.SecretsNamespacer, image *string, registryLocation, namespace, registrySecretPrefix string) (map[string]string, error) {
	privateRegistrySecretInterface := kubeClient.Secrets(namespace)
	registryEnv := make(map[string]string)
	var regSecretData map[string]string
	var err error
	if registryLocation == "off-cluster" {
		regSecretData, err = getDetailsFromRegistrySecret(privateRegistrySecretInterface, registrySecret)
		if err != nil {
			return nil, err
		}
		for key, value := range regSecretData {
			registryEnv["DRYCC_REGISTRY_"+strings.ToUpper(key)] = value
		}
		if registryEnv["DRYCC_REGISTRY_ORGANIZATION"] != "" {
			*image = registryEnv["DRYCC_REGISTRY_ORGANIZATION"] + "/" + *image
		}
		if registryEnv["DRYCC_REGISTRY_HOSTNAME"] != "" {
			*image = registryEnv["DRYCC_REGISTRY_HOSTNAME"] + "/" + *image
		}
	}
	return registryEnv, nil
}
