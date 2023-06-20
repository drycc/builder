package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
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
	FnCreate func(*v1.Secret) (*corev1.Secret, error)
	FnUpdate func(*v1.Secret) (*corev1.Secret, error)
}

// Get is the interface definition.
func (f *FakeSecret) Get(_ context.Context, name string, _ metav1.GetOptions) (*v1.Secret, error) {
	return f.FnGet(name)
}

// Delete is the interface definition.
func (f *FakeSecret) Delete(context.Context, string, metav1.DeleteOptions) error {
	return nil
}

// Create is the interface definition.
func (f *FakeSecret) Create(_ context.Context, secret *v1.Secret, _ metav1.CreateOptions) (*v1.Secret, error) {
	return f.FnCreate(secret)
}

// Update is the interface definition.
func (f *FakeSecret) Update(_ context.Context, secret *v1.Secret, _ metav1.UpdateOptions) (*v1.Secret, error) {
	return f.FnUpdate(secret)
}

// List is the interface definition.
func (f *FakeSecret) List(context.Context, metav1.ListOptions) (*v1.SecretList, error) {
	return &v1.SecretList{}, nil
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
func (f *FakeSecret) Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*v1.Secret, error) {
	return &v1.Secret{}, nil
}

func (f *FakeSecret) Apply(context.Context, *applyconfigurationscorev1.SecretApplyConfiguration, metav1.ApplyOptions) (result *v1.Secret, err error) {
	return &v1.Secret{}, nil
}

// FakeSecretsNamespacer is a mock function that can be swapped in for an
// (k8s.io/kubernetes/pkg/client/unversioned).SecretsNamespacer, so you can unit test you code
//type FakeSecretsNamespacer struct {
//	Fn func(string) typedcorev1.SecretInterface
//}

type FakeSecretsGetter struct {
	Fn func(string) typedcorev1.SecretInterface
}

// Secrets is the interface definition.Secret
func (f *FakeSecretsGetter) Secrets(namespace string) typedcorev1.SecretInterface {
	return f.Fn(namespace)
}
