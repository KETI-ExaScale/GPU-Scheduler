package resourceinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// check node count during filtering stage (golbal var)
var AvailableNodeCount = new(int)

// node total information.
type NodeInfo struct {
	NodeName           string
	Node               corev1.Node
	Pods               []*corev1.Pod
	AdditionalResource []string
	NodeScore          int  //default 0
	IsFiltered         bool //if filtered true; else false
	IsGPUNode          bool //if GPUnode true; else false
	AvailableGPUCount  int  //get number of available gpu count; default totalGPUCount
	NodeMetric         *NodeMetric
	GPUMetrics         []*GPUMetric
}

// each node metric
type NodeMetric struct {
	NodeCPU       string
	NodeMemory    string
	TotalGPUCount int
	GPU_UUID      []string
}

// each GPU metric
type GPUMetric struct {
	GPUName        string
	UUID           string
	MPSIndex       int
	GPUPower       int
	GPUMemoryTotal int
	GPUMemoryFree  int
	GPUMemoryUsed  int
	GPUTemperature int
	GPUScore       int
	IsFiltered     bool
}

type Pod struct {
	Pod                *corev1.Pod
	RequestedResource  *Resource
	ExpectedResource   *ExResource
	AdditionalResource []string
	IsGPUPod           bool
}

type Resource struct {
	MilliCPU         int
	Memory           int
	EphemeralStorage int
	GPUMPS           int
	GPUMemory        int
}

//예상 리소스 요청량
type ExResource struct {
	ExMilliCPU  int
	ExMemory    int
	ExGPUMemory int
}

func (n *NodeInfo) FilterNode() {
	n.IsFiltered = true
	*AvailableNodeCount--
}

func (g *GPUMetric) FilterGPU(n *NodeInfo) {
	g.IsFiltered = true
	n.AvailableGPUCount--
}

//return whether the node is master or not
func IsMaster(node corev1.Node) bool {
	if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
		return true
	}
	return false
}

//newly add failedCount 1
func PatchPodAnnotationFailCount(pod corev1.Pod) ([]byte, error) {
	patchAnnotations := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": {
			"failedCount": "1",
		}}}
	return json.Marshal(patchAnnotations)
}

//update failedCount ++1
func UpdatePodAnnotationFailCount(pod *corev1.Pod) error {
	failedCount, _ := strconv.Atoi(pod.ObjectMeta.Annotations["failedCount"])
	pod.ObjectMeta.Annotations["failedCount"] = strconv.Itoa(failedCount + 1)
	return nil
}

//if scheduling failed before true; else false
func IsFailedScheduling(pod *corev1.Pod) bool {
	if failedCount := pod.ObjectMeta.Annotations["failedCount"]; failedCount == "" {
		return false
	} else {
		return true
	}
}

//notice scheduling failed
func FailedScheduling(pod *corev1.Pod) error {
	if IsFailedScheduling(pod) {
		UpdatePodAnnotationFailCount(pod)
	} else {
		host_config, _ := rest.InClusterConfig()
		host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

		patchedAnnotationBytes, err := PatchPodAnnotationFailCount(*pod)
		if err != nil {
			return fmt.Errorf("failed to generate patched fs annotations,reason: %v", err)
		}

		_, err = host_kubeClient.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
		if err != nil {
			fmt.Println("FailedScheduling patch error: ", err)
		}
	}
	return nil
}

func NewResource() *Resource {
	return &Resource{
		MilliCPU:         0,
		Memory:           0,
		EphemeralStorage: 0,
		GPUMPS:           0,
		//GPUMemory:        0,
	}
}

//예상 자원 사용량 -> 없는값 0으로 통일
func NewExResource() *ExResource {
	return &ExResource{
		ExMilliCPU:  0,
		ExMemory:    0,
		ExGPUMemory: 0,
	}
}

func GetNewPodInfo(newPod *corev1.Pod) *Pod {
	isGPUPod := false
	res := NewResource()
	exres := NewExResource() //예상 자원 사용량, 현재 X
	additionalResource := make([]string, 0)

	for _, container := range newPod.Spec.Containers {
		GPUMPSLimit := container.Resources.Limits["keti.com/mpsgpu"]
		if GPUMPSLimit.String() != "" {
			temp, _ := strconv.Atoi(GPUMPSLimit.String())
			res.GPUMPS += res.GPUMPS + temp
		}
		for rName, rQuant := range container.Resources.Requests {
			switch rName {
			case corev1.ResourceCPU:
				res.MilliCPU += int(rQuant.MilliValue())
			case corev1.ResourceMemory:
				res.Memory += int(rQuant.MilliValue())
			case corev1.ResourceEphemeralStorage:
				res.EphemeralStorage += int(rQuant.MilliValue())
			default:
				// Casting from ResourceName to stirng because rName is ResourceName type
				resourceName := string(rName)
				additionalResource = append(additionalResource, resourceName)
			}
		}
	}

	if res.GPUMPS != 0 {
		isGPUPod = true
	}

	return &Pod{
		Pod:                newPod,
		RequestedResource:  res,
		ExpectedResource:   exres,
		AdditionalResource: additionalResource,
		IsGPUPod:           isGPUPod,
	}
}

type SchedulingResult struct {
	BestNode  *NodeInfo
	NodeScore int
	BestGPU   string
}

func newResult() SchedulingResult {
	return SchedulingResult{
		BestNode:  nil,
		NodeScore: 0,
		BestGPU:   "",
	}
}

var isTrue bool = true

func GetBestNodeAneGPU(nodeInfoList []*NodeInfo, requestedGPU int) SchedulingResult {
	result := newResult()

	result.BestNode = nodeInfoList[0]

	if requestedGPU == 2 {
		result.BestGPU += nodeInfoList[0].GPUMetrics[0].UUID + ","
		result.BestGPU += nodeInfoList[0].GPUMetrics[1].UUID
	} else {
		if isTrue {
			result.BestGPU += nodeInfoList[0].GPUMetrics[0].UUID
			isTrue = false
		} else {
			result.BestGPU += nodeInfoList[0].GPUMetrics[0].UUID
			isTrue = true
		}
	}

	return result
}
