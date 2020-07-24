package k8s

import (
	"time"
	"context"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	resyncPeriod = 30 * time.Second
)
type StoreToPodLister struct {
	cache.Store
}

//PodWatcher is a struct which holds the return values of (k8s.io/kubernetes/pkg/controller/framework).NewIndexerInformer together.
type PodWatcher struct {
	Store      StoreToPodLister
	Controller cache.Controller
}

//NewPodWatcher creates a new BuildPodWatcher useful to list the pods using a cache which gets updated based on the watch func.
func NewPodWatcher(c kubernetes.Interface, ns string) *PodWatcher {
	pw := &PodWatcher{}

	pw.Store.Store, pw.Controller = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc:  podListFunc(c, ns),
			WatchFunc: podWatchFunc(c, ns),
		},
		&v1.Pod{},
		resyncPeriod,
		cache.ResourceEventHandlerFuncs{},
		cache.Indexers{},
	)
	return pw
}

func podListFunc(c kubernetes.Interface, ns string) func(options metav1.ListOptions) (runtime.Object, error) {
	return func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	}
}

func podWatchFunc(c kubernetes.Interface, ns string) func(options metav1.ListOptions) (watch.Interface, error) {
	return func(opts metav1.ListOptions) (watch.Interface, error) {
		return c.CoreV1().Pods(ns).Watch(context.TODO(), metav1.ListOptions{})
	}
}
