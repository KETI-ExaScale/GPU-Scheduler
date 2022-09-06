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

	var clusterInfoList = make(map[string]*ClusterInfo)
	kubeConfigPath, err := findKubeConfig()
	if err != nil {
		fmt.Println("<error> findKubeConfig error-", err)
		return &ClusterCache{
			MyClusterName:   "",
			MyClusterInfo:   myClusterInfo,
			ClusterInfoList: nil,
			Available:       false,
		}, err
	}

	fmt.Println("kubeConfigPath: ", kubeConfigPath)

	files, err := ioutil.ReadDir(kubeConfigPath)
	if err != nil {
		fmt.Println("<error> Read Kubeconfig Path error-", err)
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
		fmt.Println("filename:", file.Name())
		kubeConfigPath_ = fmt.Sprintf("%v/%v", kubeConfigPath, file.Name())
		fmt.Println("path:", kubeConfigPath_)

		kubeConfig, err := clientcmd.LoadFromFile(kubeConfigPath_)
		if err != nil {
			fmt.Println("<error> load from file error-", err)
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
				fmt.Println("<error> BuildConfigFromFlags error-", err)
				clusterInfo.Avaliable = false
				clusterInfoList[name] = clusterInfo
				continue
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				fmt.Println("<error> NewForConfig error-", err)
				clusterInfo.Avaliable = false
				clusterInfo.Config = config
				clusterInfoList[name] = clusterInfo
				continue
			}

			clusterInfo.Config = config
			clusterInfo.Clientset = clientset
			clusterInfo.ClusterIP = cluster.Server
			clusterInfoList[name] = clusterInfo
			cnt++

			fmt.Println("-cluster name: ", name)
			fmt.Println("-ClusterIP ", cluster.Server)
			fmt.Println("---")
		}
	}

	return &ClusterCache{
		MyClusterName:       myClusterName,
		MyClusterInfo:       myClusterInfo,
		ClusterInfoList:     clusterInfoList,
		Available:           true,
		AvailableClusterCnt: cnt,
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
