package resourceinfo

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mitchellh/go-homedir"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterCache struct {
	MyClusterName       string
	MyClusterInfo       *ClusterInfo
	ClusterInfoList     map[string]*ClusterInfo
	Available           bool
	AvailableClusterCnt int
	FilteredCluster     []string
}

type ClusterInfo struct {
	Config    *rest.Config
	Clientset *kubernetes.Clientset
	ClusterIP string
	Avaliable bool
}

func NewClusterCache() (*ClusterCache, error) {
	hostConfig, _ := rest.InClusterConfig()
	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)
	myClusterInfo := NewClusterInfo()
	myClusterInfo.Config = hostConfig
	myClusterInfo.Clientset = hostKubeClient
	myClusterInfo.ClusterIP = hostConfig.Host
	var filteredCluster []string

	var clusterInfoList = make(map[string]*ClusterInfo)
	kubeConfigPath, err := findKubeConfig()
	if err != nil {
		KETI_LOG_L3(fmt.Sprintf("[error] findKubeConfig error: %s", err))
		return &ClusterCache{
			MyClusterName:   "",
			MyClusterInfo:   myClusterInfo,
			ClusterInfoList: nil,
			Available:       false,
		}, err
	}

	files, err := ioutil.ReadDir(kubeConfigPath)
	if err != nil {
		KETI_LOG_L3(fmt.Sprintf("[error] Read Kubeconfig Path error: %s", err))
		return &ClusterCache{
			MyClusterName:   "",
			MyClusterInfo:   myClusterInfo,
			ClusterInfoList: nil,
			Available:       false,
		}, err
	}

	cnt := 0
	myClusterName := ""

	for _, file := range files {
		if file.Name() == "cache" {
			continue
		}

		kubeConfigPath_ := ""
		kubeConfigPath_ = fmt.Sprintf("%v/%v", kubeConfigPath, file.Name())

		kubeConfig, err := clientcmd.LoadFromFile(kubeConfigPath_)
		if err != nil {
			KETI_LOG_L3(fmt.Sprintf("[error] load from file error: %s", err))
			continue
		}

		clusters := kubeConfig.Clusters
		currentContext := kubeConfig.CurrentContext
		currentCluster := kubeConfig.Contexts[currentContext].Cluster

		if file.Name() == "config" { //이름이 바뀔수도 있으니 다른 방법 생각
			myClusterName = currentCluster
			continue
		}

		for name, cluster := range clusters {
			clusterInfo := NewClusterInfo()

			config, err := clientcmd.BuildConfigFromFlags(cluster.Server, kubeConfigPath_)
			if err != nil {
				KETI_LOG_L3(fmt.Sprintf("[error] BuildConfigFromFlags error: %s", err))
				clusterInfo.Avaliable = false
				clusterInfoList[name] = clusterInfo
				filteredCluster = append(filteredCluster, name)
				continue
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				KETI_LOG_L3(fmt.Sprintf("[error] NewForConfig error: %s", err))
				clusterInfo.Avaliable = false
				clusterInfo.Config = config
				clusterInfoList[name] = clusterInfo
				filteredCluster = append(filteredCluster, name)
				continue
			}

			clusterInfo.Config = config
			clusterInfo.Clientset = clientset
			clusterInfo.ClusterIP = cluster.Server
			clusterInfoList[name] = clusterInfo
			cnt++
		}
	}

	return &ClusterCache{
		MyClusterName:       myClusterName,
		MyClusterInfo:       myClusterInfo,
		ClusterInfoList:     clusterInfoList,
		Available:           true,
		AvailableClusterCnt: cnt,
		FilteredCluster:     filteredCluster,
	}, nil
}

func findKubeConfig() (string, error) {
	env := os.Getenv("KUBECONFIG")
	if env != "" {
		return env, nil
	}
	path, err := homedir.Expand("/root/.kube")
	if err != nil {
		return "", err
	}
	return path, nil
}

func NewClusterInfo() *ClusterInfo {
	return &ClusterInfo{
		Config:    nil,
		Clientset: nil,
		ClusterIP: "",
		Avaliable: true,
	}
}

func (cc ClusterCache) DumpClusterInfo() {
	KETI_LOG_L1("\n-----:: Dump cluster Info Cache ::-----")
	KETI_LOG_L1(fmt.Sprintf("-total joined cluster count: %d", len(cc.ClusterInfoList)+1)) //joined + mycluster

	for cn, ci := range cc.ClusterInfoList {
		KETI_LOG_L1(fmt.Sprintf("[clusterName : %s]", cn))
		if ci.Avaliable {
			KETI_LOG_L1(fmt.Sprintf("-cluster ip: %s", ci.ClusterIP))
		} else {
			KETI_LOG_L1(">unavailable cluster")
		}
	}
}
