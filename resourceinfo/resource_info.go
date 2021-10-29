package resourceinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"sort"
	"strconv"
	"strings"

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
	NodeScore          int   //default 0
	IsFiltered         bool  //if filtered true; else false
	AvailableGPUCount  int64 //get number of available gpu count; default totalGPUCount
	NodeMetric         *NodeMetric
	GPUMetrics         []*GPUMetric
	AvailableResource  *TempResource
	CapacityResource   *TempResource
	GRPCHost           string
}

// each node metric
type NodeMetric struct {
	NodeCPU       int64
	NodeMemory    int64
	TotalGPUCount int64
	GPU_UUID      []string
}

// each GPU metric
type GPUMetric struct {
	GPUName        string
	UUID           string
	MPSIndex       int64
	GPUPower       int64
	GPUMemoryTotal int64
	GPUMemoryFree  int64
	GPUMemoryUsed  int64
	GPUTemperature int64
	GPUScore       int
	IsFiltered     bool
}

type Pod struct {
	Pod                *corev1.Pod
	RequestedResource  *Resource
	ExpectedResource   *ExResource
	AdditionalResource []string
}

type TempResource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
}

type Resource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
	GPUMPS           int64
	GPUMemory        int64 //아직 요청 X
}

//예상 리소스 요청량
type ExResource struct {
	ExMilliCPU  int64
	ExMemory    int64
	ExGPUMemory int64
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
// func IsMaster(node corev1.Node) bool {
// 	if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
// 		return true
// 	}
// 	return false
// }

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

func NewResource() *Resource {
	return &Resource{
		MilliCPU:         0,
		Memory:           0,
		EphemeralStorage: 0,
		GPUMPS:           0,
		GPUMemory:        0,
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

func NewTempResource() *TempResource {
	return &TempResource{
		MilliCPU:         0,
		Memory:           0,
		EphemeralStorage: 0,
	}
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

type SchedulingResult struct {
	BestNode   string
	TotalScore int
	BestGPU    string
}

func newResult() SchedulingResult {
	return SchedulingResult{
		BestNode:   "",
		TotalScore: 0,
		BestGPU:    "",
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
