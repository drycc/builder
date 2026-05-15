package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// StoreToPodLister is a wrapper around cache.Store that provides methods to list pods.
type StoreToPodLister struct {
	cache.Store
}

// PodWatcher is a struct which holds the return values of (k8s.io/kubernetes/pkg/controller/framework).NewIndexerInformer together.
type PodWatcher struct {
	Store      StoreToPodLister
	Controller cache.Controller
}

// List returns a list of pods that match the given label selector.
func (s *StoreToPodLister) List(selector labels.Selector) (pods []*v1.Pod, err error) {
	// TODO: it'd be great to just call
	// s.Pods(api.NamespaceAll).List(selector), however then we'd have to
	// remake the list.Items as a []*api.Pod. So leave this separate for
	// now.
	for _, m := range s.Store.List() {
		pod := m.(*v1.Pod)
		if selector.Matches(labels.Set(pod.Labels)) {
			pods = append(pods, pod)
		}
	}
	return pods, nil
}

// NewPodWatcher creates a new PodWatcher backed by a cache populated from a
// label-scoped list/watch. A tight labelSelector keeps the WatchList initial
// events stream small enough for the apiserver to deliver the terminating
// bookmark within client-go's timeout; an empty string watches the namespace.
func NewPodWatcher(c kubernetes.Clientset, ns, labelSelector string) *PodWatcher {
	pw := &PodWatcher{}

	pw.Store.Store, pw.Controller = cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: &cache.ListWatch{
			ListFunc:  podListFunc(c, ns, labelSelector),
			WatchFunc: podWatchFunc(c, ns, labelSelector),
		},
		ObjectType: &v1.Pod{},
		Handler:    cache.ResourceEventHandlerFuncs{},
	})
	return pw
}

// podListFunc and podWatchFunc preserve any options injected by the reflector
// (ResourceVersion, AllowWatchBookmarks, SendInitialEvents, ...) and only
// override LabelSelector. Overwriting options with a fresh metav1.ListOptions{}
// here would break the WatchList path used by client-go >= v0.35.
func podListFunc(c kubernetes.Clientset, ns, labelSelector string) func(options metav1.ListOptions) (runtime.Object, error) {
	return func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = labelSelector
		return c.CoreV1().Pods(ns).List(context.TODO(), options)
	}
}

func podWatchFunc(c kubernetes.Clientset, ns, labelSelector string) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(options metav1.ListOptions) (watch.Interface, error) {
		options.LabelSelector = labelSelector
		return c.CoreV1().Pods(ns).Watch(context.TODO(), options)
	}
}
