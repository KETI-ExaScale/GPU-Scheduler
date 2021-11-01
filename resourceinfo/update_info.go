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
	"encoding/json"
	"fmt"
	"gpu-scheduler/config"
	"math"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	if config.Debugg {
		fmt.Println("[step 0] Get Nodes/GPU MultiMetric")
	}
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

	// c, err := client.NewHTTPClient(client.HTTPConfig{
	// 	Addr: config.URL,
	// })
	// if err != nil {
	// 	fmt.Println("Error creatring influx", err.Error())
	// }
	// defer c.Close()

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
		newNodeMetric := GetNodeMetric(node.Name, MCIP)
		availableGPUCount := newNodeMetric.TotalGPUCount
		newGPUMetrics = GetGPUMetrics(newNodeMetric.GPU_UUID, MCIP)

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

func (n *NodeInfo) FilterNode() {
	n.IsFiltered = true
	*AvailableNodeCount--
}

func (g *GPUMetric) FilterGPU(n *NodeInfo) {
	g.IsFiltered = true
	n.AvailableGPUCount--
}

//return whether the node is GPUNode or not
func IsNonGPUNode(node corev1.Node) bool {
	if _, ok := node.Labels["gpu"]; ok {
		return false
	}
	return true
}

//newly add failedCount 1
func PatchPodAnnotationFailCount(pod *corev1.Pod, count int) ([]byte, error) {
	patchAnnotations := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": {
			"failedCount": strconv.Itoa(count),
		}}}
	return json.Marshal(patchAnnotations)
}

//notice scheduling failed
func FailedScheduling(pod *corev1.Pod) error {
	if count, ok := pod.Labels["schedulingCount"]; ok {
		c, _ := strconv.Atoi(count)
		patchedAnnotationBytes, err := PatchPodAnnotationFailCount(pod, c+1)
		if err != nil {
			return fmt.Errorf("failed to generate patched fs annotations,reason: %v", err)
		}
		host_config, _ := rest.InClusterConfig()
		host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

		_, err = host_kubeClient.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
		if err != nil {
			fmt.Println("FailedScheduling patch error: ", err)
		}
	} else {
		patchedAnnotationBytes, err := PatchPodAnnotationFailCount(pod, 1)
		if err != nil {
			return fmt.Errorf("failed to generate patched fs annotations,reason: %v", err)
		}
		host_config, _ := rest.InClusterConfig()
		host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

		_, err = host_kubeClient.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
		if err != nil {
			fmt.Println("FailedScheduling patch error: ", err)
		}
	}

	return nil
}

func GetNewPodInfo(newPod *corev1.Pod) *Pod {
	res := NewResource()
	exres := NewExResource() //예상 자원 사용량, 현재 X
	additionalResource := make([]string, 0)

	for _, container := range newPod.Spec.Containers {
		GPUMPSLimit := container.Resources.Limits["keti.com/mpsgpu"]
		if GPUMPSLimit.String() != "" {
			temp, _ := strconv.Atoi(GPUMPSLimit.String())
			res.GPUMPS += res.GPUMPS + int64(temp)
		}
		for rName, rQuant := range container.Resources.Requests {
			switch rName {
			case corev1.ResourceCPU:
				res.MilliCPU += int64(rQuant.MilliValue())
			case corev1.ResourceMemory:
				res.Memory += int64(rQuant.MilliValue())
			case corev1.ResourceEphemeralStorage:
				res.EphemeralStorage += int64(rQuant.MilliValue())
			default:
				// Casting from ResourceName to stirng because rName is ResourceName type
				resourceName := string(rName)
				additionalResource = append(additionalResource, resourceName)
			}
		}
	}

	return &Pod{
		Pod:                newPod,
		RequestedResource:  res,
		ExpectedResource:   exres,
		AdditionalResource: additionalResource,
	}
}

func GetBestNodeAneGPU(nodeInfoList []*NodeInfo, requestedGPU int64) SchedulingResult {
	result := newResult()

	for _, node := range nodeInfoList {
		totalScore, bestGPU := getTotalScore(node, requestedGPU)
		if result.TotalScore < totalScore {
			result.BestNode = node.NodeName
			result.BestGPU = bestGPU
			result.TotalScore = totalScore
		}
	}

	//fmt.Println("[[result]] ", result)
	return result
}

func getTotalScore(node *NodeInfo, requestedGPU int64) (int, string) {
	weight, _ := exec.Command("cat", "/tmp/node-gpu-score-weight").Output()
	nodeWeight, _ := strconv.ParseFloat(strings.Split(string(weight), " ")[0], 64)
	gpuWeight, _ := strconv.ParseFloat(strings.Split(string(weight), " ")[1], 64)
	//fmt.Println("[[NodeScore]] ", node.NodeScore)
	totalGPUScore, bestGPU := getTotalGPUScore(node.GPUMetrics, requestedGPU)
	totalScore := math.Round(float64(node.NodeScore)*nodeWeight + float64(totalGPUScore)*gpuWeight)

	//fmt.Println("[[totalScoreResult]] ", totalScore, bestGPU)
	return int(totalScore), bestGPU
}

func getTotalGPUScore(gpuMetrics []*GPUMetric, requestedGPU int64) (int, string) {
	totalGPUScore, bestGPU := float64(0), ""

	sort.Slice(gpuMetrics, func(i, j int) bool {
		return gpuMetrics[i].GPUScore > gpuMetrics[j].GPUScore
	})

	//fmt.Println("[[requestedGPU]] ", requestedGPU)

	bestGPUMetrics := gpuMetrics[:requestedGPU]
	for _, gpu := range bestGPUMetrics {
		bestGPU += gpu.UUID + ","
		//fmt.Println("[[GPUScore]] ", gpu.UUID, gpu.GPUScore)
		totalGPUScore += float64(gpu.GPUScore) * float64(1/float64(requestedGPU))

	}
	totalGPUScore, bestGPU = math.Round(totalGPUScore), strings.Trim(bestGPU, ",")

	//fmt.Println("[[NodetotalGPUScoreResult]] ", totalGPUScore, bestGPU)

	return int(totalGPUScore), bestGPU
}
