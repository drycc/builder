package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/api/core/v1"
)

// NamespaceLister is a (k8s.io/kubernetes/pkg/client/unversioned).NamespaceInterface compatible
// interface which only has the List function. It's used in places that only need List to make
// them easier to test and more easily swappable with other implementations
// (should the need arise).
//
// Example usage:
//
//	var nsl NamespaceLister
//	nsl = kubeClient.Namespaces()
type NamespaceLister interface {
	List(opts metav1.ListOptions) (*v1.NamespaceList, error)
}
