package resourceinfo

import (
	corev1 "k8s.io/api/core/v1"
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

func NewTempResource() *TempResource {
	return &TempResource{
		MilliCPU:         0,
		Memory:           0,
		EphemeralStorage: 0,
	}
}

type Resource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
	GPUMPS           int64
	GPUMemory        int64 //아직 요청 X
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

//예상 리소스 요청량
type ExResource struct {
	ExMilliCPU  int64
	ExMemory    int64
	ExGPUMemory int64
}

//예상 자원 사용량 -> 없는값 0으로 통일
func NewExResource() *ExResource {
	return &ExResource{
		ExMilliCPU:  0,
		ExMemory:    0,
		ExGPUMemory: 0,
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
