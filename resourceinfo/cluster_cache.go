package resourceinfo

import (
	"fmt"
	"os/exec"
	"sync"
)

type ClusterCache struct {
	// period time.Duration
	// ttl                    time.Duration
	mu              sync.RWMutex
	MyClusterName   string
	ClusterInfoList map[string]*ClusterInfo
	// GPUMemoryMostInCluster int64
	TotalNodeCount int
	// AvailableNodeCount     int
}

func NewClusterCache() *ClusterCache {
	totalNodeCount := 0
	clusterInfoList, myClusterName := NewClusterInfo()
	for _, ci := range clusterInfoList {
		totalNodeCount += ci.NodeCount
	}

	return &ClusterCache{
		MyClusterName:   myClusterName,
		ClusterInfoList: clusterInfoList,
		TotalNodeCount:  totalNodeCount,
	}
}

type ClusterInfo struct {
	ClusterName string
	NodeCount   int
}

func NewClusterInfo() (map[string]*ClusterInfo, string) {
	var (
		myClusterName string
		cluster0      string
		cluster1      string
	)

	clusterInfoList := make(map[string]*ClusterInfo)

	myClusterName_, err := exec.Command("cat", "/gpu-scheduler-clusterinfo/my-cluster-name").Output()
	if err == nil {
		myClusterName = string(myClusterName_)
		fmt.Println("myclustername: ", myClusterName)
	}

	cluster0_, err := exec.Command("cat", "/gpu-scheduler-clusterinfo/cluster0").Output()
	if err == nil {
		cluster0 = string(cluster0_)
		fmt.Println("cluster0_: gpu1" /*, cluster0_*/)
	}

	cluster1_, err := exec.Command("cat", "/gpu-scheduler-clusterinfo/cluster1").Output()
	if err == nil {
		cluster1 = string(cluster1_)
		fmt.Println("cluster1_: kubernetes" /*, cluster1_*/)
	}

	if cluster0 != myClusterName {
		ci0 := &ClusterInfo{
			ClusterName: cluster0,
			NodeCount:   1,
		}
		clusterInfoList[cluster0] = ci0
	}

	if cluster1 != myClusterName {
		ci1 := &ClusterInfo{
			ClusterName: cluster1,
			NodeCount:   1,
		}
		clusterInfoList[cluster1] = ci1
	}

	// fmt.Println("[Dump Cluster Info Cache]")
	// for cName, cInfo := range clusterInfoList {
	// 	fmt.Println("(1) cluster name {", cName, "}")
	// 	fmt.Print("(2) num of nodes: ", cInfo.NodeCount, "\n")
	// }

	return clusterInfoList, myClusterName
}
