package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	applyconfigurationscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// FakeSecret is a mock function that can be swapped in for
// (k8s.io/kubernetes/pkg/client/unversioned).SecretsInterface,
// so you can unit test your code.
type FakeSecret struct {
	FnGet    func(string) (*corev1.Secret, error)
	FnCreate func(*corev1.Secret) (*corev1.Secret, error)
	FnUpdate func(*corev1.Secret) (*corev1.Secret, error)
}

// Get is the interface definition.
func (f *FakeSecret) Get(_ context.Context, name string, _ metav1.GetOptions) (*corev1.Secret, error) {
	return f.FnGet(name)
}

// Delete is the interface definition.
func (f *FakeSecret) Delete(context.Context, string, metav1.DeleteOptions) error {
	return nil
}

// Create is the interface definition.
func (f *FakeSecret) Create(_ context.Context, secret *corev1.Secret, _ metav1.CreateOptions) (*corev1.Secret, error) {
	return f.FnCreate(secret)
}

// Update is the interface definition.
func (f *FakeSecret) Update(_ context.Context, secret *corev1.Secret, _ metav1.UpdateOptions) (*corev1.Secret, error) {
	return f.FnUpdate(secret)
}

// List is the interface definition.
func (f *FakeSecret) List(context.Context, metav1.ListOptions) (*corev1.SecretList, error) {
	return &corev1.SecretList{}, nil
}

// Watch is the interface definition.
func (f *FakeSecret) Watch(context.Context, metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}

// DeleteCollection is the interface definition.
func (f *FakeSecret) DeleteCollection(context.Context, metav1.DeleteOptions, metav1.ListOptions) error {
	return nil
}

// Patch is the interface definition.
func (f *FakeSecret) Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*corev1.Secret, error) {
	return &corev1.Secret{}, nil
}

// Apply is the interface definition for applying a secret configuration.
func (f *FakeSecret) Apply(context.Context, *applyconfigurationscorev1.SecretApplyConfiguration, metav1.ApplyOptions) (result *corev1.Secret, err error) {
	return &corev1.Secret{}, nil
}

// FakeSecretsNamespacer is a mock function that can be swapped in for an
// (k8s.io/kubernetes/pkg/client/unversioned).SecretsNamespacer, so you can unit test you code
//type FakeSecretsNamespacer struct {
//	Fn func(string) typedcorev1.SecretInterface
//}

// FakeSecretsGetter is a mock function that can be swapped in for an
// (k8s.io/kubernetes/pkg/client/unversioned).SecretsNamespacer, so you can unit test you code
type FakeSecretsGetter struct {
	Fn func(string) typedcorev1.SecretInterface
}

// Secrets is the interface definition.Secret
func (f *FakeSecretsGetter) Secrets(namespace string) typedcorev1.SecretInterface {
	return f.Fn(namespace)
}
