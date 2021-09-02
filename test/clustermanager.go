package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ClusterManager struct {
	Host_config     *rest.Config
	Host_kubeClient *kubernetes.Clientset
	//Mutex	*sync.Mutex
}

func NewClusterManager() *ClusterManager {
	//mutex := &sync.Mutex{}
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

	cm := &ClusterManager{
		Host_config:     host_config,
		Host_kubeClient: host_kubeClient,
		//Mutex:	mutex,
	}
	return cm
}
