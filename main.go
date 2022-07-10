package main

import (
	"context"
	"fmt"

	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Config struct {
	//Handler Handler
	Resource  string
	Namespace string
}

type Event struct {
	key          string
	eventType    string
	resourceType string
}

func main() {
	conf := Config{Namespace: "default"}

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
	}

	//if conf.Resource == "ConfigMap" {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().ConfigMaps(conf.Namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().ConfigMaps(conf.Namespace).Watch(context.TODO(), options)
			},
		},
		&api_v1.ConfigMap{},
		0, //Skip resync
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			var newEvent Event
			newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.eventType = "create"
			newEvent.resourceType = "configMap"
			fmt.Println(newEvent)
		},
		UpdateFunc: func(old, new interface{}) {
			var newEvent Event
			newEvent.key, err = cache.MetaNamespaceKeyFunc(old)
			newEvent.eventType = "update"
			newEvent.resourceType = "configMap"
			fmt.Println(newEvent)
		},
		DeleteFunc: func(obj interface{}) {
			var newEvent Event
			newEvent.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			newEvent.eventType = "delete"
			newEvent.resourceType = "configMap"
			fmt.Println(newEvent)
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	informer.Run(stopCh)

	//wait.Until(nil, time.Second, stopCh)
	//}
}
