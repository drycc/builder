package k8s

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// FakeSecret is a mock function that can be swapped in for
// (k8s.io/kubernetes/pkg/client/unversioned).SecretsInterface,
// so you can unit test your code.
type FakeSecret struct {
	FnGet    func(string) (*v1.Secret, error)
	FnCreate func(*v1.Secret) (*v1.Secret, error)
	FnUpdate func(*v1.Secret) (*v1.Secret, error)
}

// Get is the interface definition.
func (f *FakeSecret) Get(name string) (*v1.Secret, error) {
	return f.FnGet(name)
}

// Delete is the interface definition.
func (f *FakeSecret) Delete(name string) error {
	return nil
}

// Create is the interface definition.
func (f *FakeSecret) Create(secret *v1.Secret) (*v1.Secret, error) {
	return f.FnCreate(secret)
}

// Update is the interface definition.
func (f *FakeSecret) Update(secret *v1.Secret) (*v1.Secret, error) {
	return f.FnUpdate(secret)
}

// List is the interface definition.
func (f *FakeSecret) List(opts metav1.ListOptions) (*v1.SecretList, error) {
	return &v1.SecretList{}, nil
}

// Watch is the interface definition.
func (f *FakeSecret) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}

// FakeSecretsNamespacer is a mock function that can be swapped in for an
// (k8s.io/kubernetes/pkg/client/unversioned).SecretsNamespacer, so you can unit test you code
type FakeSecretsNamespacer struct {
	Fn func(string) corev1.SecretInterface
}

// Secrets is the interface definition.
func (f *FakeSecretsNamespacer) Secrets(namespace string) corev1.SecretInterface {
	return f.Fn(namespace)
}
