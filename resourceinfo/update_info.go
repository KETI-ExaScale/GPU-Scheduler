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

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetNodes() (*corev1.NodeList, error) {
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)
	nodeList, _ := host_kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	return nodeList, nil
}

func GetPods() (*corev1.PodList, error) {
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)
	podList, _ := host_kubeClient.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})

	return podList, nil
}

func NodeUpdate(nodeInfoList []*NodeInfo, nodeMetricList []*NodeMetric) ([]*NodeInfo, []*NodeMetric, error) {
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

	ip := "influxdb.gpu.svc.cluster.local"
	port := "8086"
	url := "http://" + ip + ":" + port
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: url,
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

		*AvailableNodeCount++

		podsInNode := make([]*corev1.Pod, 0)

		for _, pod := range pods.Items {
			if strings.Compare(pod.Spec.NodeName, node.Name) == 0 {
				podsInNode = append(podsInNode, &pod)
			}
		}

		// Get Affinity
		node_affinity := make(map[string]string)

		for key, value := range node.Labels {
			switch key {
			case "failure-domain.beta.kubernetes.io/region":
				if _, ok := node_affinity["region"]; !ok {
					node_affinity["region"] = value
				}

			case "failure-domain.beta.kubernetes.io/zone":
				if _, ok := node_affinity["zone"]; !ok {
					node_affinity["zone"] = value
				}
			}
		}

		// make new Node
		newNodeInfo := &NodeInfo{
			NodeName:  node.Name,
			Node:      node,
			Pods:      podsInNode,
			Affinity:  node_affinity,
			NodeScore: 0,
		}
		nodeInfoList = append(nodeInfoList, newNodeInfo)

		q := client.Query{
			Command:  fmt.Sprintf("SELECT last(*) FROM metric where NodeName=%s", node.Name),
			Database: "multimetric",
		}
		aaa := make(map[string]string)
		if response, err := c.Query(q); err == nil && response.Error() == nil {
			fmt.Println(response.Results[0].Series[0])
			aaa = response.Results[0].Series[0].Values
			NodeCPU := aaa[0]
			NodeMemory := aaa[1]
			GPUCount := aaa[2]
			UUID := aaa[3]
			var myData [][]interface{} = make([][]interface{}, len(response.Results[0].Series[0].Values))
			for i, d := range response.Results[0].Series[0].Values {
				myData[i] = d
			}

			fmt.Println("", myData[0]) //first element in slice
			fmt.Println("", myData[0][0])
			fmt.Println("", myData[0][1])

		}

		newNodeMetric := &NodeMetric{
			NodeName:   node.Name,
			NodeCPU:    aaa[0],
			NodeMemory: NodeMemory,
			GPUCount:   GPUCount,
			UUID:       UUID,
		}
		nodeMetricList = append(nodeMetricList, newNodeMetric)

	}

	return nodeInfoList, nodeMetricList, nil
}
