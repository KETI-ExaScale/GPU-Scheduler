// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resourceinfo

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gpu-scheduler/config"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//Get Nodes in Cluster
func GetNodes() (*corev1.NodeList, error) {
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)
	nodeList, _ := host_kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	return nodeList, nil
}

//Get Pods in Cluster
func GetPods() (*corev1.PodList, error) {
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)
	podList, _ := host_kubeClient.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})

	return podList, nil
}

//Update NodeInfo
func NodeUpdate(nodeInfoList []*NodeInfo) ([]*NodeInfo, error) {
	fmt.Println("[step 0] Get Nodes/GPU MultiMetric")
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: config.URL,
	})
	if err != nil {
		fmt.Println("Error creatring influx", err.Error())
	}
	defer c.Close()

	pods, _ := host_kubeClient.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	nodes, _ := host_kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	for _, node := range nodes.Items {

		if IsMaster(node) {
			continue
		}

		isGPUNode, availableGPUCount := false, 0
		var newGPUMetrics []*GPUMetric

		CountUpAvailableNodeCount()

		podsInNode := getPodsInNode(pods, node.Name)
		newNodeMetric := getNodeMetric(c, node.Name)

		if newNodeMetric.TotalGPUCount != 0 {
			isGPUNode = true
			availableGPUCount = newNodeMetric.TotalGPUCount
			newGPUMetrics = getGPUMetrics(c, newNodeMetric.GPU_UUID)
		}

		// make new Node
		newNodeInfo := &NodeInfo{
			NodeName:          node.Name,
			Node:              node,
			Pods:              podsInNode,
			NodeScore:         0,
			IsFiltered:        false,
			IsGPUNode:         isGPUNode,
			AvailableGPUCount: availableGPUCount,
			NodeMetric:        newNodeMetric,
			GPUMetrics:        newGPUMetrics,
		}

		nodeInfoList = append(nodeInfoList, newNodeInfo)
	}

	return nodeInfoList, nil
}

func CountUpAvailableNodeCount() {
	*AvailableNodeCount++
}

func getPodsInNode(pods *corev1.PodList, nodeName string) []*corev1.Pod {
	podsInNode := make([]*corev1.Pod, 0)
	for _, pod := range pods.Items {
		if strings.Compare(pod.Spec.NodeName, nodeName) == 0 {
			podsInNode = append(podsInNode, &pod)
		}
	}
	return podsInNode
}

func getNodeMetric(c client.Client, nodeName string) *NodeMetric {
	q := client.Query{
		Command:  fmt.Sprintf("SELECT last(*) FROM metric where NodeName='%s'", nodeName),
		Database: "multimetric",
	}

	response, err := c.Query(q)
	if err != nil || response.Error() != nil {
		fmt.Println("InfluxDB error: ", err)
		return nil
	}

	myNodeMetric := response.Results[0].Series[0].Values[0]

	totalGPUCount, _ := strconv.Atoi(fmt.Sprintf("%s", myNodeMetric[1]))
	nodeCPU := fmt.Sprintf("%s", myNodeMetric[2])
	nodeMemory := fmt.Sprintf("%s", myNodeMetric[3])
	uuids := stringToArray(myNodeMetric[4].(string))

	fmt.Println(" |NodeMetric|", totalGPUCount, nodeCPU, nodeMemory, uuids)
	return &NodeMetric{
		NodeCPU:       nodeCPU,
		NodeMemory:    nodeMemory,
		TotalGPUCount: totalGPUCount,
		GPU_UUID:      uuids,
	}
}

//'[abc abc]' : string -> ['abc' 'abc'] : []string
func stringToArray(str string) []string {
	str = strings.Trim(str, "[]")
	return strings.Split(str, " ")
}

func getGPUMetrics(c client.Client, uuids []string) []*GPUMetric {
	var tempGPUMetrics []*GPUMetric

	for _, uuid := range uuids {
		q := client.Query{
			Command:  fmt.Sprintf("SELECT last(*) FROM metric where UUID='%s'", uuid),
			Database: "gpumap",
		}

		response, err := c.Query(q)
		if err != nil || response.Error() != nil {
			fmt.Println("InfluxDB error: ", err)
			return nil
		}

		myGPUMetric := response.Results[0].Series[0].Values[0]

		gpuName := fmt.Sprintf("%s", myGPUMetric[1])
		mpsIndex, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[2]))
		gpuPower, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[3]))
		gpuMemoryTotal, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[6]))
		gpuMemoryFree, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[4]))
		gpuMemoryUsed, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[7]))
		gpuTemperature, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[8]))

		newGPUMetric := &GPUMetric{
			GPUName:        gpuName,
			UUID:           uuid,
			MPSIndex:       mpsIndex,
			GPUPower:       gpuPower,
			GPUMemoryTotal: gpuMemoryTotal,
			GPUMemoryFree:  gpuMemoryFree,
			GPUMemoryUsed:  gpuMemoryUsed,
			GPUTemperature: gpuTemperature,
			IsFiltered:     false,
			GPUScore:       0,
		}

		fmt.Println(" |GPUMetric|", newGPUMetric)
		tempGPUMetrics = append(tempGPUMetrics, newGPUMetric)
	}

	return tempGPUMetrics
}
