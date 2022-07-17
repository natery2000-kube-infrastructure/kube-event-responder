package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	resourceName string
	action       string
	resourceType string
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
			newEvent.resourceName, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.action = "create"
			newEvent.resourceType = "configMap"
			eventChan <- newEvent
		},
		UpdateFunc: func(old, new interface{}) {
			var newEvent Event
			newEvent.resourceName, err = cache.MetaNamespaceKeyFunc(old)
			newEvent.action = "update"
			newEvent.resourceType = "configMap"
			eventChan <- newEvent
		},
		DeleteFunc: func(obj interface{}) {
			var newEvent Event
			newEvent.resourceName, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			newEvent.action = "delete"
			newEvent.resourceType = "configMap"
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
