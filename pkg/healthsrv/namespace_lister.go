package healthsrv

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	//"k8s.io/apimachinery/pkg/labels"
	//"k8s.io/apimachinery/pkg/fields"
	//"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceLister is an (*k8s.io/kubernetes/pkg/client/unversioned).Client compatible interface
// that provides just the ListBuckets cross-section of functionality. It can also be implemented
// for unit tests.
type NamespaceLister interface {
	// List lists all namespaces that are selected by the given label and field selectors.
    List(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error)
}

type emptyNamespaceLister struct{}

func (n emptyNamespaceLister) List(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error) {
	return &corev1.NamespaceList{}, nil
}

type errNamespaceLister struct {
	err error
}

func (e errNamespaceLister) List(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error) {
	return nil, e.err
}


// listNamespaces calls nl.List(...) and sends the results back on the various given channels.
// This func is intended to be run in a goroutine and communicates via the channels it's passed.
//
// On success, it passes the namespace list on succCh, and on failure, it passes the error on
// errCh. At most one of {succCh, errCh} will be sent on. If stopCh is closed, no pending or
// future sends will occur.
func listNamespaces(nl NamespaceLister, succCh chan<- *corev1.NamespaceList, errCh chan<- error, stopCh <-chan struct{}) {
	nsList, err := nl.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		select {
		case errCh <- err:
		case <-stopCh:
		}
		return
	}
	select {
	case succCh <- nsList:
	case <-stopCh:
		return
	}
}
