package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	apps_v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	api_v1 "k8s.io/api/core/v1"
	ext_v1beta1 "k8s.io/api/extensions/v1beta1"
	rbac_v1beta1 "k8s.io/api/rbac/v1beta1"
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
	resourceName string
	action       string
	resourceType string
	raw          interface{}
}

func main() {
	conf := Config{Namespace: "default"}
	eventChan := make(chan Event)
	var handlers []Handler

	kubectlCommandHandler := KubectlCommandHandler{
		command: "kubectl get pods",
		HandlerBase: HandlerBase{
			action:       "update",
			resourceName: "default/streaming-couchdb-configmap",
		},
	}
	handlers = append(handlers, kubectlCommandHandler)

	printlnHandler := PrintlnHandler{
		printString: "hello world",
		HandlerBase: HandlerBase{
			action:       "update",
			resourceName: "default/streaming-couchdb-configmap",
		},
	}
	handlers = append(handlers, printlnHandler)

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
			kubeobj, _, _ := informer.GetIndexer().GetByKey(newEvent.key)
			objectMeta := GetObjectMetaData(kubeobj)
			newEvent.resourceName, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.action = "create"
			newEvent.resourceType = "configMap"
			newEvent.raw = objectMeta
			eventChan <- newEvent
		},
		UpdateFunc: func(old, new interface{}) {
			var newEvent Event
			newEvent.key, err = cache.MetaNamespaceKeyFunc(new)
			kubeobj, _, _ := informer.GetIndexer().GetByKey(newEvent.key)
			objectMeta := GetObjectMetaData(kubeobj)
			newEvent.resourceName, err = cache.MetaNamespaceKeyFunc(new)
			newEvent.action = "update"
			newEvent.resourceType = "configMap"
			newEvent.raw = objectMeta
			eventChan <- newEvent
		},
		DeleteFunc: func(obj interface{}) {
			var newEvent Event
			newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
			kubeobj, _, _ := informer.GetIndexer().GetByKey(newEvent.key)
			objectMeta := GetObjectMetaData(kubeobj)
			newEvent.resourceName, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			newEvent.action = "delete"
			newEvent.resourceType = "configMap"
			newEvent.raw = objectMeta
			eventChan <- newEvent
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	go informer.Run(stopCh)

	cache.WaitForCacheSync(stopCh, informer.HasSynced)

	fmt.Println("starting listening")
	worker(eventChan, handlers)
}

func worker(eventChan <-chan Event, handlers []Handler) {
	for event := range eventChan {
		fmt.Println("event receieved", event)
		for _, handler := range handlers {
			handler.Handle(event)
		}
	}
}

type Handler interface {
	Handle(e Event)
}

type HandlerBase struct {
	action       string
	resourceName string
}

type KubectlCommandHandler struct {
	HandlerBase
	command string
}

func (handler KubectlCommandHandler) Handle(e Event) {
	if !(handler.action == e.action && handler.resourceName == e.resourceName) {
		return
	}

	var cmd *exec.Cmd
	if strings.Contains(handler.command, " ") {
		parts := strings.Split(handler.command, " ")
		cmd = exec.Command(parts[0], parts[1:]...)
	} else {
		cmd = exec.Command(handler.command)
	}
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

type PrintlnHandler struct {
	HandlerBase
	printString string
}

func (handler PrintlnHandler) Handle(e Event) {
	if !(handler.action == e.action && handler.resourceName == e.resourceName) {
		return
	}
	fmt.Println(handler.printString, e)
}

// GetObjectMetaData returns metadata of a given k8s object
func GetObjectMetaData(obj interface{}) (objectMeta meta_v1.ObjectMeta) {

	switch object := obj.(type) {
	case *apps_v1.Deployment:
		objectMeta = object.ObjectMeta
	case *api_v1.ReplicationController:
		objectMeta = object.ObjectMeta
	case *apps_v1.ReplicaSet:
		objectMeta = object.ObjectMeta
	case *apps_v1.DaemonSet:
		objectMeta = object.ObjectMeta
	case *api_v1.Service:
		objectMeta = object.ObjectMeta
	case *api_v1.Pod:
		objectMeta = object.ObjectMeta
	case *batch_v1.Job:
		objectMeta = object.ObjectMeta
	case *api_v1.PersistentVolume:
		objectMeta = object.ObjectMeta
	case *api_v1.Namespace:
		objectMeta = object.ObjectMeta
	case *api_v1.Secret:
		objectMeta = object.ObjectMeta
	case *ext_v1beta1.Ingress:
		objectMeta = object.ObjectMeta
	case *api_v1.Node:
		objectMeta = object.ObjectMeta
	case *rbac_v1beta1.ClusterRole:
		objectMeta = object.ObjectMeta
	case *api_v1.ServiceAccount:
		objectMeta = object.ObjectMeta
	case *api_v1.Event:
		objectMeta = object.ObjectMeta
	}
	return objectMeta
}
