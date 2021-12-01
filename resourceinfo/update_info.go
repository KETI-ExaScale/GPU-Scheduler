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
	"log"
	"math"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

//Update NodeInfo
func NodeUpdate(nodeInfoList []*NodeInfo) ([]*NodeInfo, error) {
	if config.Debugg {
		fmt.Println("[step 0] Get Nodes/GPU MultiMetric")
		fmt.Println("<Sending gRPC request>")
	}

	pods, _ := config.Host_kubeClient.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	nodes, _ := config.Host_kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	for _, node := range nodes.Items {

		//Skip NonGPUNode
		if IsNonGPUNode(node) {
			continue
		}

		allocatableres := NewTempResource() //temp
		var newGPUMetrics []*GPUMetric

		CountUpAvailableNodeCount()

		podsInNode, host := getPodsInNode(pods, node.Name)
		newNodeMetric := GetNodeMetric(node.Name, host)
		newGPUMetrics = GetGPUMetrics(newNodeMetric.GPU_UUID, host)
		//imageStates := addNodeImageStates(node)

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

		fmt.Println("node allocatable : ", allocatableres)

		// make new Node
		newNodeInfo := &NodeInfo{
			NodeName:            node.Name,
			Node:                node,
			Pods:                podsInNode,
			AvailableGPUCount:   newNodeMetric.TotalGPUCount,
			NodeScore:           0,
			IsFiltered:          false,
			NodeMetric:          newNodeMetric,
			GPUMetrics:          newGPUMetrics,
			AllocatableResource: allocatableres,
			GRPCHost:            host,
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
			if strings.HasPrefix(pod.Name, "keti-gpu-metric-collector") {
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

		_, err = config.Host_kubeClient.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
		if err != nil {
			fmt.Println("FailedScheduling patch error: ", err)
		}
	} else {
		patchedAnnotationBytes, err := PatchPodAnnotationFailCount(pod, 1)
		if err != nil {
			return fmt.Errorf("failed to generate patched fs annotations,reason: %v", err)
		}

		_, err = config.Host_kubeClient.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
		if err != nil {
			fmt.Println("FailedScheduling patch error: ", err)
		}
	}

	return nil
}

func GetNewPodInfo(newPod *corev1.Pod) *Pod {
	res := NewResource()
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
				res.Memory += int64(rQuant.Value())
			case corev1.ResourceEphemeralStorage:
				res.EphemeralStorage += int64(rQuant.Value())
			default:
				// Casting from ResourceName to stirng because rName is ResourceName type
				resourceName := string(rName)
				additionalResource = append(additionalResource, resourceName)
			}
		}
	}

	fmt.Println("pod info : ", res)

	return &Pod{
		Pod:                newPod,
		RequestedResource:  res,
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

	return result
}

func getTotalScore(node *NodeInfo, requestedGPU int64) (int, string) {

	totalGPUScore, bestGPU := getTotalGPUScore(node.GPUMetrics, requestedGPU)
	totalScore := math.Round(float64(node.NodeScore)*config.NodeWeight + float64(totalGPUScore)*config.GPUWeight)

	return int(totalScore), bestGPU
}

func getTotalGPUScore(gpuMetrics []*GPUMetric, requestedGPU int64) (int, string) {
	totalGPUScore, bestGPU := float64(0), ""

	sort.Slice(gpuMetrics, func(i, j int) bool {
		return gpuMetrics[i].GPUScore > gpuMetrics[j].GPUScore
	})

	bestGPUMetrics := gpuMetrics[:requestedGPU]
	for _, gpu := range bestGPUMetrics {
		bestGPU += gpu.UUID + ","
		totalGPUScore += float64(gpu.GPUScore) * float64(1/float64(requestedGPU))

	}
	totalGPUScore, bestGPU = math.Round(totalGPUScore), strings.Trim(bestGPU, ",")

	return int(totalGPUScore), bestGPU
}

func IsThereAnyNode(newPod *Pod) bool {
	if *AvailableNodeCount == 0 {
		FailedScheduling(newPod.Pod)
		message := fmt.Sprintf("pod (%s) failed to fit in any node", newPod.Pod.ObjectMeta.Name)
		log.Println(message)
		event := MakeNoNodeEvent(newPod, message)
		PostEvent(event)
		return false
	}
	return true
}

func UpdatePolicy() {
	weightPolicy, _ := exec.Command("cat", "/tmp/node-gpu-score-weight").Output()
	nodeWeight, _ := strconv.ParseFloat(strings.Split(string(weightPolicy), " ")[0], 64)
	gpuWeight, _ := strconv.ParseFloat(strings.Split(string(weightPolicy), " ")[1], 64)
	reSchedulePolicy, _ := exec.Command("cat", "/tmp/pod-re-schedule-permit").Output()
	reSchedule, _ := strconv.ParseBool(string(reSchedulePolicy))

	config.NodeWeight, config.GPUWeight, config.ReSchedule = nodeWeight, gpuWeight, reSchedule
}

// func addNodeImageStates(node *corev1.Node) map[string]*ImageState {
// 	newImageStates := make(map[string]*ImageState)

// 	for _, image := range node.Status.Images {
// 		for _, name := range image.Names {
// 			// update the entry in imageStates
// 			state, ok := imageStates[name]
// 			if !ok {
// 				state = &imageState{
// 					size:  image.SizeBytes,
// 					nodes: sets.NewString(node.Name),
// 				}
// 				cache.imageStates[name] = state
// 			} else {
// 				state.nodes.Insert(node.Name)
// 			}
// 			// create the imageStateSummary for this image
// 			if _, ok := newSum[name]; !ok {
// 				newSum[name] = cache.createImageStateSummary(state)
// 			}
// 		}
// 	}

// 	return newImageStates
// }
