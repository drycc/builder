package gitreceive

import (
	"errors"
	"testing"

	"github.com/arschles/assert"
	"github.com/drycc/builder/pkg/k8s"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

const (
	testSecret     = "test-secret"
	dryccNamespace = "drycc"
)

func TestGetDetailsFromRegistrySecretErr(t *testing.T) {
	expectedErr := errors.New("get secret error")
	getter := &k8s.FakeSecret{
		FnGet: func(string) (*api.Secret, error) {
			return &api.Secret{}, expectedErr
		},
	}
	_, err := getDetailsFromRegistrySecret(getter, testSecret)
	assert.Err(t, err, expectedErr)
}

func TestGetDetailsFromRegistrySecretSuccess(t *testing.T) {
	data := map[string][]byte{"test": []byte("test")}
	expectedData := map[string]string{"test": "test"}
	secret := api.Secret{Data: data}
	getter := &k8s.FakeSecret{
		FnGet: func(string) (*api.Secret, error) {
			return &secret, nil
		},
	}
	secretData, err := getDetailsFromRegistrySecret(getter, testSecret)
	assert.NoErr(t, err)
	assert.Equal(t, secretData, expectedData, "secret data")
}

func TestGetRegistryDetailsOffclusterErr(t *testing.T) {
	expectedErr := errors.New("get secret error")
	getter := &k8s.FakeSecret{
		FnGet: func(string) (*api.Secret, error) {
			return &api.Secret{}, expectedErr
		},
	}

	kubeClient := &k8s.FakeSecretsNamespacer{
		Fn: func(string) client.SecretsInterface {
			return getter
		},
	}
	image := "test-image"
	_, err := getRegistryDetails(kubeClient, &image, "off-cluster", dryccNamespace, "private-registry")
	assert.Err(t, err, expectedErr)
}

func TestGetRegistryDetailsOffclusterSuccess(t *testing.T) {
	data := map[string][]byte{"organization": []byte("kmala"), "hostname": []byte("quay.io")}
	expectedData := map[string]string{"DRYCC_REGISTRY_HOSTNAME": "quay.io", "DRYCC_REGISTRY_ORGANIZATION": "kmala"}
	expectedImage := "quay.io/kmala/test-image"
	secret := api.Secret{Data: data}
	getter := &k8s.FakeSecret{
		FnGet: func(string) (*api.Secret, error) {
			return &secret, nil
		},
	}

	kubeClient := &k8s.FakeSecretsNamespacer{
		Fn: func(string) client.SecretsInterface {
			return getter
		},
	}
	image := "test-image"
	regDetails, err := getRegistryDetails(kubeClient, &image, "off-cluster", dryccNamespace, "private-registry")
	assert.NoErr(t, err)
	assert.Equal(t, expectedData, regDetails, "registry details")
	assert.Equal(t, expectedImage, image, "image")
}
