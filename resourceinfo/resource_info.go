package resourceinfo

import (
	corev1 "k8s.io/api/core/v1"
)

// check node count during filtering stage (golbal var)
//var AvailableNodeCount = new(int)
var NodeCount *_NodeCount

type _NodeCount struct {
	NodeAvailable int
}

func InitNodeCount() *_NodeCount {
	return &_NodeCount{
		NodeAvailable: 0,
	}
}

func (nc *_NodeCount) CountUpNodeAvailableCount() {
	nc.NodeAvailable++
}

func (nc *_NodeCount) CountDownNodeAvailableCount() {
	nc.NodeAvailable--
}

var NodeInfoList []*NodeInfo

// node total information.
type NodeInfo struct {
	NodeName            string
	Node                corev1.Node
	Pods                []*corev1.Pod
	AvailableGPUCount   int64 //get number of available gpu count; default totalGPUCount
	NodeScore           int   //default 0
	IsFiltered          bool  //if filtered true; else false
	IsGPUNode           bool
	NodeMetric          *NodeMetric
	GPUMetrics          []*GPUMetric
	AllocatableResource *TempResource
	GRPCHost            string
	//AdditionalResource []string
	//ImageStates map[string]*ImageState
}

func InitNodeInfoList() []*NodeInfo {
	return nil
}

func (n *NodeInfo) FilterNode() {
	n.IsFiltered = true
	NodeCount.NodeAvailable--
}

// each node metric
type NodeMetric struct {
	MilliCPUTotal int64
	MilliCPUUsed  int64
	MemoryTotal   int64
	MemoryUsed    int64
	StorageTotal  int64
	StorageUsed   int64
	TotalGPUCount int64
	GPU_UUID      []string
	MaxGPUMemory  int64
}

// each GPU metric
type GPUMetric struct {
	GPUName        string
	UUID           string
	GPUIndex       int64
	GPUPower       int64
	GPUMemoryTotal int64
	GPUMemoryFree  int64
	GPUMemoryUsed  int64
	GPUTemperature int64
	GPUScore       int
	IsFiltered     bool
	PodCount       int64
}

func (g *GPUMetric) FilterGPU(n *NodeInfo) {
	g.IsFiltered = true
	n.AvailableGPUCount--
}

var NewPod *Pod

//newly added Pod
type Pod struct {
	Pod                *corev1.Pod
	RequestedResource  *Resource
	IsGPUPod           bool
	GPUMemoryLimit     int64
	GPUMemoryRequest   int64
	AdditionalResource []string
}

func InitNewPod() *Pod {
	return &Pod{
		Pod:                nil,
		RequestedResource:  NewResource(),
		IsGPUPod:           false,
		GPUMemoryLimit:     0,
		GPUMemoryRequest:   0,
		AdditionalResource: make([]string, 0),
	}
}

type TempResource struct {
	MilliCPU int64
	Memory   int64
	Storage  int64
}

func NewTempResource() *TempResource {
	return &TempResource{
		MilliCPU: 0,
		Memory:   0,
		Storage:  0,
	}
}

type Resource struct {
	MilliCPU  int64
	Memory    int64
	Storage   int64
	GPUCount  int64
	GPUMemory int64
}

func NewResource() *Resource {
	return &Resource{
		MilliCPU:  0,
		Memory:    0,
		Storage:   0,
		GPUCount:  0,
		GPUMemory: 1,
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
		TotalScore: -1,
		BestGPU:    "",
	}
}

type ImageState struct {
	// Size of the image
	Size int64
	// Used to track how many nodes have this image
	NumNodes int
}
