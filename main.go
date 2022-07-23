package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	apps_v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	api_v1 "k8s.io/api/core/v1"
	ext_v1beta1 "k8s.io/api/extensions/v1beta1"
	rbac_v1beta1 "k8s.io/api/rbac/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	//eventChan := make(chan Event)
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
		config = &rest.Config{
			Host: "https://" + net.JoinHostPort(os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")),
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
			BearerToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6IlFhVUY4aFJ4M002NTQ4SWxRc3drS3pWUGtFc2FFVXJWX0JpbUMyZW1scWsifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiXSwiZXhwIjoxNjU4NjA2MTUzLCJpYXQiOjE2NTg2MDI1NTMsImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJkZWZhdWx0Iiwic2VydmljZWFjY291bnQiOnsibmFtZSI6Imt1YmUtZXZlbnQtcmVzcG9uZGVyLXNlcnZpY2VhY2NvdW50IiwidWlkIjoiOTdjZTAxOGQtMDM4Ni00NDRkLThiYjAtMWM0MDMxM2YwMzk3In19LCJuYmYiOjE2NTg2MDI1NTMsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDpkZWZhdWx0Omt1YmUtZXZlbnQtcmVzcG9uZGVyLXNlcnZpY2VhY2NvdW50In0.TP1AGuRQWUm0HBrMDRQxjZdCxsbu6XcybgP0UtoMJCzUS46p13i3khjJVBw5kjEPxRjJTqm3cp4q0eYiDEP0B-WXEpcnvfZRK0zzCHB10jl2C5f1gMHl4jUpRPo7MMrOFI6P6EGR5H3uPhuMUtUaSnkvQMXe54AwlSoQH9MihzOAwUu1nvMCputJXbXDuTAdnsjq-Znadzr-UqfFEiUiAc1pqI187bhM8NuzL25yQln7_JGuYf77wt5M79l6Mq0XuZUaARU0l_Wz4aJEcAzEew6eGUU13o4YZKRLnJlcRVNaDP2vCabcQqDU_a0RpeepRLWXylpZxSZfLTHRUgT_Lg",
		}
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
	}

	var api = kubeClient.CoreV1().ConfigMaps(conf.Namespace)
	configMaps, err := api.List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	resourceVersion := configMaps.ListMeta.ResourceVersion

	watcher, err := api.Watch(context.TODO(), meta_v1.ListOptions{ResourceVersion: resourceVersion})
	if err != nil {
		fmt.Println(err)
	}

	ch := watcher.ResultChan()

	fmt.Println("starting listen")
	for {
		event := <-ch
		configMap, _ := event.Object.(*api_v1.ConfigMap)
		jsonConfig, _ := json.Marshal(configMap)
		fmt.Println(string(jsonConfig))
	}

	// informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
	// 	AddFunc: func(obj interface{}) {
	// 		var newEvent Event
	// 		newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
	// 		kubeobj, _, _ := informer.GetIndexer().GetByKey(newEvent.key)
	// 		objectMeta := GetObjectMetaData(kubeobj)
	// 		newEvent.resourceName, err = cache.MetaNamespaceKeyFunc(obj)
	// 		newEvent.action = "create"
	// 		newEvent.resourceType = "configMap"
	// 		newEvent.raw = objectMeta
	// 		eventChan <- newEvent
	// 	},
	// 	UpdateFunc: func(old, new interface{}) {
	// 		var newEvent Event
	// 		newEvent.key, err = cache.MetaNamespaceKeyFunc(new)
	// 		kubeobj, _, _ := informer.GetIndexer().GetByKey(newEvent.key)
	// 		objectMeta := GetObjectMetaData(kubeobj)
	// 		newEvent.resourceName, err = cache.MetaNamespaceKeyFunc(new)
	// 		newEvent.action = "update"
	// 		newEvent.resourceType = "configMap"
	// 		newEvent.raw = objectMeta
	// 		eventChan <- newEvent
	// 	},
	// 	DeleteFunc: func(obj interface{}) {
	// 		var newEvent Event
	// 		newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
	// 		kubeobj, _, _ := informer.GetIndexer().GetByKey(newEvent.key)
	// 		objectMeta := GetObjectMetaData(kubeobj)
	// 		newEvent.resourceName, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	// 		newEvent.action = "delete"
	// 		newEvent.resourceType = "configMap"
	// 		newEvent.raw = objectMeta
	// 		eventChan <- newEvent
	// 	},
	// })

	// stopCh := make(chan struct{})
	// defer close(stopCh)
	// go informer.Run(stopCh)

	// cache.WaitForCacheSync(stopCh, informer.HasSynced)

	// fmt.Println("starting listening")
	// worker(eventChan, handlers)
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
