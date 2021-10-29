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
	"strings"

	"gpu-scheduler/config"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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

		//Skip NonGPUNode
		if IsNonGPUNode(node) {
			continue
		}

		capacityres := NewTempResource()    //temp
		allocatableres := NewTempResource() //temp
		var newGPUMetrics []*GPUMetric

		CountUpAvailableNodeCount()

		podsInNode, MCIP := getPodsInNode(pods, node.Name)
		newNodeMetric := GetNodeMetric(c, node.Name, MCIP)
		availableGPUCount := newNodeMetric.TotalGPUCount
		newGPUMetrics = GetGPUMetrics(c, newNodeMetric.GPU_UUID, MCIP)

		//현재 매트릭콜렉터 말고 따로 자원량 수집 중(메트릭에서 단위 맞춰서 가져올예정)
		for rName, rQuant := range node.Status.Capacity {
			switch rName {
			case corev1.ResourceCPU:
				capacityres.MilliCPU = rQuant.MilliValue()
			case corev1.ResourceMemory:
				capacityres.Memory = rQuant.Value()
			case corev1.ResourceEphemeralStorage:
				capacityres.EphemeralStorage = rQuant.Value()
			default:
				// Casting from ResourceName to stirng because rName is ResourceName type
			}
		}
		for rName, rQuant := range node.Status.Allocatable {
			switch rName {
			case corev1.ResourceCPU:
				allocatableres.MilliCPU = rQuant.MilliValue()
			case corev1.ResourceMemory:
				allocatableres.Memory = rQuant.Value()
			case corev1.ResourceEphemeralStorage:
				allocatableres.EphemeralStorage = rQuant.Value()
			default:
				// Casting from ResourceName to stirng because rName is ResourceName type
			}
		}

		// make new Node
		newNodeInfo := &NodeInfo{
			NodeName:          node.Name,
			Node:              node,
			Pods:              podsInNode,
			NodeScore:         0,
			IsFiltered:        false,
			AvailableGPUCount: availableGPUCount,
			NodeMetric:        newNodeMetric,
			GPUMetrics:        newGPUMetrics,
			AvailableResource: allocatableres,
			CapacityResource:  capacityres,
			GRPCHost:          MCIP,
		}

		nodeInfoList = append(nodeInfoList, newNodeInfo)
	}

	return nodeInfoList, nil
}

func CountUpAvailableNodeCount() {
	*AvailableNodeCount++
}

func getPodsInNode(pods *corev1.PodList, nodeName string) ([]*corev1.Pod, string) {
	podsInNode, MCIP := make([]*corev1.Pod, 0), ""

	for _, pod := range pods.Items {
		if strings.Compare(pod.Spec.NodeName, nodeName) == 0 {
			podsInNode = append(podsInNode, &pod)
			if strings.HasPrefix(pod.Name, "gpu-metric-collector") {
				MCIP = pod.Status.PodIP
			}
		}
	}

	return podsInNode, MCIP
}
