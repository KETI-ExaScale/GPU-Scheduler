package resourceinfo

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"strconv"

// 	corev1 "k8s.io/api/core/v1"
// 	"k8s.io/client-go/kubernetes"
// 	"k8s.io/client-go/rest"

// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/types"
// )

// var ClusterInfo *_ClusterInfo

// //const variable
// const (
// 	N             = float64(4) //노드 스코어링 단계 수
// 	SchedulerName = "gpu-scheduler"
// 	Policy1       = "node-gpu-score-weight"
// 	Policy2       = "pod-re-schedule-permit"
// )

// // max gpu memory in cluster for scoring
// var (
// 	GPUMemoryTotalMost = int64(0)
// )

// type _ClusterInfo struct {
// 	HostConfig     *rest.Config
// 	HostKubeClient *kubernetes.Clientset
// 	Nodes          *corev1.NodeList
// 	Pods           *corev1.PodList
// }

// func NewClusterInfo() *_ClusterInfo {
// 	hostConfig, _ := rest.InClusterConfig()
// 	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)

// 	return &_ClusterInfo{
// 		HostConfig:     hostConfig,
// 		HostKubeClient: hostKubeClient,
// 		Nodes:          nil,
// 		Pods:           nil,
// 	}
// }

// func (c *_ClusterInfo) UpdateClusterInfo() {
// 	c.Pods, _ = c.HostKubeClient.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
// 	c.Nodes, _ = c.HostKubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
// }

// type SchedulingPolicy struct {
// 	NodeWeight              float64
// 	GPUWeight               float64
// 	ReSchedulePermit        bool
// 	LeastAllocatedPodPrefer bool
// }

// type ImageStateSummary struct {
// 	// Size of the image
// 	Size int64
// 	// Used to track how many nodes have this image
// 	NumNodes int
// }

// // node total information.
// type NodeInfo struct {
// 	NodeName            string
// 	Node                corev1.Node
// 	Pods                *corev1.PodList
// 	TotalGPUCount       int64
// 	IsGPUNode           bool
// 	NodeMetric          *NodeMetric
// 	GPUMetrics          []*GPUMetric
// 	AllocatableResource *TempResource //삭제예정
// 	GRPCHost            string
// 	PodsWithAffinity    []*PodInfo
// 	//AdditionalResource []string
// 	ImageStates map[string]*ImageStateSummary
// }

// // each node metric
// type NodeMetric struct {
// 	MilliCPUTotal int64
// 	MilliCPUUsed  int64
// 	MemoryTotal   int64
// 	MemoryUsed    int64
// 	StorageTotal  int64
// 	StorageUsed   int64
// 	TotalGPUCount int64
// 	GPU_UUID      []string
// 	MaxGPUMemory  int64
// }

// func NewNodeMetric() *NodeMetric {
// 	return &NodeMetric{
// 		MilliCPUTotal: 0,
// 		MilliCPUUsed:  0,
// 		MemoryTotal:   0,
// 		MemoryUsed:    0,
// 		StorageTotal:  0,
// 		StorageUsed:   0,
// 		TotalGPUCount: 0,
// 		GPU_UUID:      nil,
// 		MaxGPUMemory:  0,
// 	}
// }

// // each GPU metric
// type GPUMetric struct {
// 	GPUName        string
// 	UUID           string
// 	GPUIndex       int64
// 	GPUPowerUsed   int64
// 	GPUPowerTotal  int64
// 	GPUMemoryTotal int64
// 	GPUMemoryFree  int64
// 	GPUMemoryUsed  int64
// 	GPUTemperature int64
// 	PodCount       int64
// 	GPUFlops       int64
// 	GPUArch        int64
// 	GPUUtil        int64
// }

// func NewGPUMetric() *GPUMetric {
// 	return &GPUMetric{
// 		GPUName:        "",
// 		UUID:           "",
// 		GPUIndex:       0,
// 		GPUPowerUsed:   0,
// 		GPUPowerTotal:  0,
// 		GPUMemoryTotal: 0,
// 		GPUMemoryFree:  0,
// 		GPUMemoryUsed:  0,
// 		GPUTemperature: 0,
// 		PodCount:       0,
// 		GPUFlops:       0,
// 		GPUArch:        0,
// 		GPUUtil:        0,
// 	}
// }

// type ImageState struct {
// 	// Size of the image
// 	Size int64
// 	// Used to track how many nodes have this image
// 	NumNodes int
// }

// //newly added Pod
// type Pod struct {
// 	Pod               *corev1.Pod
// 	RequestedResource *Resource
// 	IsGPUPod          bool
// }

// func NewPod() *Pod {
// 	return &Pod{
// 		Pod:               nil,
// 		RequestedResource: NewResource(),
// 		IsGPUPod:          false,
// 	}
// }

// func (p *Pod) InitPod() {
// 	p.Pod = nil
// 	p.RequestedResource = NewResource()
// 	p.IsGPUPod = false
// }

// type TempResource struct {
// 	MilliCPU int64
// 	Memory   int64
// 	Storage  int64
// }

// func NewTempResource() *TempResource {
// 	return &TempResource{
// 		MilliCPU: 0,
// 		Memory:   0,
// 		Storage:  0,
// 	}
// }

// type Resource struct {
// 	MilliCPU         int64
// 	Memory           int64
// 	Storage          int64
// 	GPUCount         int
// 	GPUMemoryLimit   int64
// 	GPUMemoryRequest int64
// }

// func NewResource() *Resource {
// 	return &Resource{
// 		MilliCPU:         0,
// 		Memory:           0,
// 		Storage:          0,
// 		GPUCount:         0,
// 		GPUMemoryLimit:   0,
// 		GPUMemoryRequest: 1,
// 	}
// }

// //newly add failedCount 1
// func patchPodAnnotationFailCount(count int) ([]byte, error) {
// 	patchAnnotations := map[string]interface{}{
// 		"metadata": map[string]map[string]string{"annotations": {
// 			"failedCount": strconv.Itoa(count),
// 		}}}
// 	return json.Marshal(patchAnnotations)
// }

// //notice scheduling failed
// func (p *Pod) FailedScheduling() error {
// 	if count, ok := p.Pod.Annotations["failedCount"]; ok {
// 		cnt, _ := strconv.Atoi(count)
// 		patchedAnnotationBytes, err := patchPodAnnotationFailCount(cnt + 1)
// 		if err != nil {
// 			return fmt.Errorf("failed to generate patched fs annotations,reason: %v", err)
// 		}

// 		_, err = ClusterInfo.HostKubeClient.CoreV1().Pods(p.Pod.Namespace).Patch(context.TODO(), p.Pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
// 		if err != nil {
// 			fmt.Println("FailedScheduling patch error: ", err)
// 		}
// 	} else {
// 		patchedAnnotationBytes, err := patchPodAnnotationFailCount(1)
// 		if err != nil {
// 			return fmt.Errorf("failed to generate patched fs annotations,reason: %v", err)
// 		}

// 		_, err = ClusterInfo.HostKubeClient.CoreV1().Pods(p.Pod.Namespace).Patch(context.TODO(), p.Pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
// 		if err != nil {
// 			fmt.Println("FailedScheduling patch error: ", err)
// 		}
// 	}

// 	return nil
// }

// func (p *Pod) IsDeleted() bool {
// 	fmt.Println("is deleted called")
// 	fmt.Println(p.Pod.Status.Phase)
// 	if t := p.Pod.Status.Phase; t == "Unknown" {
// 		fmt.Println("<pod deleted> ", p.Pod.Name)
// 		return true
// 	}
// 	return false
// }

// func (p *Pod) MakeNoNodeEvent(message string) *corev1.Event {
// 	return &corev1.Event{
// 		Count:          1,
// 		Message:        message,
// 		Reason:         "FailedScheduling",
// 		LastTimestamp:  metav1.Now(),
// 		FirstTimestamp: metav1.Now(),
// 		Type:           "Warning",
// 		Source: corev1.EventSource{
// 			Component: SchedulerName,
// 		},
// 		InvolvedObject: corev1.ObjectReference{
// 			Kind:      "Pod",
// 			Name:      p.Pod.Name,
// 			Namespace: "default",
// 			UID:       p.Pod.UID,
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			GenerateName: p.Pod.Name + "-",
// 			Name:         p.Pod.Name,
// 		},
// 	}
// }

// func (p *Pod) MakeBindEvent(message string) *corev1.Event {
// 	return &corev1.Event{
// 		Count:          1,
// 		Message:        message,
// 		Reason:         "Scheduled",
// 		LastTimestamp:  metav1.Now(),
// 		FirstTimestamp: metav1.Now(),
// 		Type:           "Normal",
// 		Source: corev1.EventSource{
// 			Component: SchedulerName,
// 		},
// 		InvolvedObject: corev1.ObjectReference{
// 			Kind:      "Pod",
// 			Name:      p.Pod.Name,
// 			Namespace: p.Pod.Namespace,
// 			UID:       p.Pod.UID,
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			GenerateName: p.Pod.Name + "-",
// 			Name:         p.Pod.Name,
// 		},
// 	}
// }

// type SchedulingResult struct {
// 	BestNode   string
// 	BestGPU    string
// 	TotalScore int
// }

// func NewSchedulingResult() *SchedulingResult {
// 	return &SchedulingResult{
// 		BestNode:   "",
// 		BestGPU:    "",
// 		TotalScore: -1,
// 	}
// }

// func (sr *SchedulingResult) InitResult() {
// 	sr.BestNode = ""
// 	sr.BestGPU = ""
// 	sr.TotalScore = -1
// }

// type PluginResult struct {
// 	AvailableNodeCount int
// 	Scores             []*Score
// }

// func NewPluginResult() *PluginResult {
// 	return &PluginResult{
// 		AvailableNodeCount: 0,
// 		Scores:             nil,
// 	}
// }

// func (pr *PluginResult) NodeCountUp() {
// 	pr.AvailableNodeCount++
// }

// func (pr *PluginResult) NodeCountDown() {
// 	pr.AvailableNodeCount--
// }

// type Score struct {
// 	NodeName          string
// 	AvailableGPUCount int
// 	IsFiltered        bool
// 	NodeScore         int
// 	GPUScores         []*GPUScore
// 	TotalGPUScore     int
// 	TotalScore        int
// 	BestGPU           string
// }

// type GPUScore struct {
// 	UUID       string
// 	IsFiltered bool
// 	GPUScore   int
// }

// func (s *Score) GPUCountUp() {
// 	s.AvailableGPUCount++
// }

// func (s *Score) GPUCountDown() {
// 	s.AvailableGPUCount--
// }

// func NewScore(nodename string) *Score {
// 	return &Score{
// 		NodeName:          nodename,
// 		AvailableGPUCount: 0,
// 		IsFiltered:        false,
// 		NodeScore:         MinNodeScore,
// 		GPUScores:         nil,
// 		TotalGPUScore:     MinGPUScore,
// 		TotalScore:        MinTotalScore,
// 		BestGPU:           "",
// 	}
// }

// func NewGPUScore(uuid string) *GPUScore {
// 	return &GPUScore{
// 		UUID:       uuid,
// 		IsFiltered: false,
// 		GPUScore:   MinGPUScore,
// 	}
// }

// func (s *Score) FilterNode() {
// 	s.IsFiltered = true
// }

// func (gs *GPUScore) FilterGPU() {
// 	gs.IsFiltered = true
// }

// const (
// 	MaxNodeScore     int = 100
// 	MinNodeScore     int = 0
// 	MaxTotalScore    int = 100 // NodeScore * NodeWeight + TotalGPUScore * GPUWeight
// 	MinTotalScore    int = 0
// 	MaxGPUScore      int = 100
// 	MinGPUScore      int = 0
// 	MaxTotalGPUScore int = 100
// 	MinTotalGPUScore int = 0
// )
