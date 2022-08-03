/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resourceinfo

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	Ns            = int(10) //노드 스코어링 단계 수
	Gs            = int(10) //GPU 스코어링 단계 수
	SchedulerName = "gpu-scheduler"
	Policy1       = "node-gpu-score-weight"
	Policy2       = "pod-re-schedule-permit"
	Policy3       = "node-reservation-permit"
	Policy4       = "nvlink-weight-percentage"
	Policy5       = "gpu-allocate-prefer"
)

const (
	MaxScore int = 100
	MinScore int = 0
)

type ActionType int64

const (
	Add    ActionType = 1 << iota // 1
	Delete                        // 10
	// UpdateNodeXYZ is only applicable for Node events.
	UpdateNodeAllocatable // 100
	UpdateNodeLabel       // 1000
	UpdateNodeTaint       // 10000
	UpdateNodeCondition   // 100000

	All ActionType = 1<<iota - 1 // 111111

	// Use the general Update type if you don't either know or care the specific sub-Update type to use.
	Update = UpdateNodeAllocatable | UpdateNodeLabel | UpdateNodeTaint | UpdateNodeCondition
)

// GVK is short for group/version/kind, which can uniquely represent a particular API resource.
type GVK string

// Constants for GVKs.
const (
	Pod                   GVK = "Pod"
	Node                  GVK = "Node"
	PersistentVolume      GVK = "PersistentVolume"
	PersistentVolumeClaim GVK = "PersistentVolumeClaim"
	Service               GVK = "Service"
	StorageClass          GVK = "storage.k8s.io/StorageClass"
	CSINode               GVK = "storage.k8s.io/CSINode"
	CSIDriver             GVK = "storage.k8s.io/CSIDriver"
	CSIStorageCapacity    GVK = "storage.k8s.io/CSIStorageCapacity"
	WildCard              GVK = "*"
)

// ClusterEvent abstracts how a system resource's state gets changed.
// Resource represents the standard API resources such as Pod, Node, etc.
// ActionType denotes the specific change such as Add, Update or Delete.
type ClusterEvent struct {
	Resource   GVK
	ActionType ActionType
	Label      string
}

// IsWildCard returns true if ClusterEvent follows WildCard semantics
func (ce ClusterEvent) IsWildCard() bool {
	return ce.Resource == WildCard && ce.ActionType == All
}

// type _ClusterInfo struct {
// 	HostConfig     *rest.Config
// 	HostKubeClient *kubernetes.Clientset
// }

// func NewClusterInfo() *_ClusterInfo {
// 	hostConfig, _ := rest.InClusterConfig()
// 	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)

// 	return &_ClusterInfo{
// 		HostConfig:     hostConfig,
// 		HostKubeClient: hostKubeClient,
// 	}
// }

type ScheduleResult struct {
	BestNode   string
	BestGPU    string
	TotalScore int
}

func NewScheduleResult() *ScheduleResult {
	return &ScheduleResult{
		BestNode:   "",
		BestGPU:    "",
		TotalScore: -1,
	}
}

func (sr *ScheduleResult) InitResult() {
	sr.BestNode = ""
	sr.BestGPU = ""
	sr.TotalScore = -1
}

type PluginResult struct {
	AvailableGPUCount int
	IsFiltered        bool
	FilteredStage     string
	NodeScore         int
	GPUScores         map[string]*GPUScore
	TotalGPUScore     int
	TotalScore        int
	BestGPU           string
}

type GPUScore struct {
	UUID          string
	IsFiltered    bool
	FilteredStage string
	GPUScore      int
	IsSelected    bool
}

func (pr *PluginResult) GPUCountUp() {
	pr.AvailableGPUCount++
}

func (pr *PluginResult) GPUCountDown() {
	pr.AvailableGPUCount--
}

func NewPluginResult() *PluginResult {
	return &PluginResult{
		AvailableGPUCount: 0,
		IsFiltered:        false,
		FilteredStage:     "",
		NodeScore:         MinScore,
		GPUScores:         make(map[string]*GPUScore),
		TotalGPUScore:     MinScore,
		TotalScore:        MinScore,
		BestGPU:           "",
	}
}

func (pr *PluginResult) InitPluginResult() {
	pr.AvailableGPUCount = 0
	pr.IsFiltered = false
	pr.FilteredStage = ""
	pr.NodeScore = MinScore
	for uuid, gpuscore := range pr.GPUScores {
		gpuscore.InitGPUScore(uuid)
	}
	pr.TotalGPUScore = MinScore
	pr.TotalScore = MinScore
	pr.BestGPU = ""
}

func NewGPUScore(uuid string) *GPUScore {
	return &GPUScore{
		UUID:          uuid,
		IsFiltered:    false,
		FilteredStage: "",
		GPUScore:      MinScore,
		IsSelected:    false,
	}
}

func (gs *GPUScore) InitGPUScore(uuid string) {
	gs.UUID = uuid
	gs.GPUScore = MinScore
	gs.FilteredStage = ""
	gs.IsFiltered = false
	gs.IsSelected = false
}

func (pr *PluginResult) FilterNode(stage string) {
	pr.IsFiltered = true
	pr.FilteredStage = stage
}

func (gs *GPUScore) FilterGPU(stage string) {
	gs.IsFiltered = true
	gs.FilteredStage = stage
}

type QueuedPodInfo struct {
	PodUID types.UID
	*PodInfo
	Timestamp               time.Time   //큐에 들어온 시간
	Attempts                int         //스케줄링 시도 횟수
	InitialAttemptTimestamp time.Time   //큐에 최초로 들어온 시간
	UnschedulablePlugins    sets.String //실패한 단계의 이름
	activate                bool
	TargetCluster           string
}

func newQueuedPodInfo(pod *corev1.Pod) *QueuedPodInfo {
	return &QueuedPodInfo{
		PodUID:                  pod.UID,
		PodInfo:                 GetNewPodInfo(pod),
		Timestamp:               time.Now(),
		Attempts:                0,
		InitialAttemptTimestamp: time.Now(),
		activate:                true,
		TargetCluster:           "",
	}
}

func (qpi *QueuedPodInfo) DeepCopy() *QueuedPodInfo {
	return &QueuedPodInfo{
		PodUID:                  qpi.PodUID,
		PodInfo:                 qpi.PodInfo.DeepCopy(),
		Timestamp:               qpi.Timestamp,
		Attempts:                qpi.Attempts,
		InitialAttemptTimestamp: qpi.InitialAttemptTimestamp,
		activate:                qpi.activate,
		TargetCluster:           qpi.TargetCluster,
	}
}

func (qpi *QueuedPodInfo) Activate() {
	qpi.Timestamp = time.Now()
	qpi.Attempts++
}

type PodInfo struct {
	Pod                        *corev1.Pod
	RequiredAffinityTerms      []AffinityTerm
	RequiredAntiAffinityTerms  []AffinityTerm
	PreferredAffinityTerms     []WeightedAffinityTerm
	PreferredAntiAffinityTerms []WeightedAffinityTerm
	ParseError                 error
	RequestedResource          *PodResource
	IsGPUPod                   bool
	ReserveNode                string
}

func newPodInfo() *PodInfo {
	return &PodInfo{
		Pod:                        nil,
		RequiredAffinityTerms:      nil,
		RequiredAntiAffinityTerms:  nil,
		PreferredAffinityTerms:     nil,
		PreferredAntiAffinityTerms: nil,
		ParseError:                 nil,
		RequestedResource:          nil,
		IsGPUPod:                   true,
		ReserveNode:                "",
	}
}

func GetNewPodInfo(pod *corev1.Pod) *PodInfo {
	podinfo := newPodInfo()
	podinfo.Update(pod)
	return podinfo
}

func (pi *PodInfo) DeepCopy() *PodInfo {
	return &PodInfo{
		Pod:                        pi.Pod.DeepCopy(),
		RequiredAffinityTerms:      pi.RequiredAffinityTerms,
		RequiredAntiAffinityTerms:  pi.RequiredAntiAffinityTerms,
		PreferredAffinityTerms:     pi.PreferredAffinityTerms,
		PreferredAntiAffinityTerms: pi.PreferredAntiAffinityTerms,
		ParseError:                 pi.ParseError,
		RequestedResource:          pi.RequestedResource,
	}
}

func (pi *PodInfo) Update(pod *corev1.Pod) {
	if pod != nil && pi.Pod != nil && pi.Pod.UID == pod.UID {
		// PodInfo includes immutable information, and so it is safe to update the pod in place if it is
		// the exact same pod
		pi.Pod = pod
		return
	}
	var preferredAffinityTerms []corev1.WeightedPodAffinityTerm
	var preferredAntiAffinityTerms []corev1.WeightedPodAffinityTerm
	if affinity := pod.Spec.Affinity; affinity != nil {
		if a := affinity.PodAffinity; a != nil {
			preferredAffinityTerms = a.PreferredDuringSchedulingIgnoredDuringExecution
		}
		if a := affinity.PodAntiAffinity; a != nil {
			preferredAntiAffinityTerms = a.PreferredDuringSchedulingIgnoredDuringExecution
		}
	}

	// Attempt to parse the affinity terms
	var parseErrs []error
	requiredAffinityTerms, err := getAffinityTerms(pod, getPodAffinityTerms(pod.Spec.Affinity))
	if err != nil {
		parseErrs = append(parseErrs, fmt.Errorf("requiredAffinityTerms: %w", err))
	}
	requiredAntiAffinityTerms, err := getAffinityTerms(pod,
		getPodAntiAffinityTerms(pod.Spec.Affinity))
	if err != nil {
		parseErrs = append(parseErrs, fmt.Errorf("requiredAntiAffinityTerms: %w", err))
	}
	weightedAffinityTerms, err := getWeightedAffinityTerms(pod, preferredAffinityTerms)
	if err != nil {
		parseErrs = append(parseErrs, fmt.Errorf("preferredAffinityTerms: %w", err))
	}
	weightedAntiAffinityTerms, err := getWeightedAffinityTerms(pod, preferredAntiAffinityTerms)
	if err != nil {
		parseErrs = append(parseErrs, fmt.Errorf("preferredAntiAffinityTerms: %w", err))
	}

	requestedResource, isGPUPod := calculatePodResource(pod)

	pi.Pod = pod
	pi.RequiredAffinityTerms = requiredAffinityTerms
	pi.RequiredAntiAffinityTerms = requiredAntiAffinityTerms
	pi.PreferredAffinityTerms = weightedAffinityTerms
	pi.PreferredAntiAffinityTerms = weightedAntiAffinityTerms
	pi.ParseError = utilerrors.NewAggregate(parseErrs)
	pi.RequestedResource = requestedResource
	pi.IsGPUPod = isGPUPod
}

func getMemory(memory string) int64 {
	if memory == "" {
		return 0
	} else {
		rQuant := resource.MustParse(memory)
		return int64(rQuant.Value())
	}
}

// AffinityTerm is a processed version of v1.PodAffinityTerm.
type AffinityTerm struct {
	Namespaces        sets.String
	Selector          labels.Selector
	TopologyKey       string
	NamespaceSelector labels.Selector
}

// Matches returns true if the pod matches the label selector and namespaces or namespace selector.
func (at *AffinityTerm) Matches(pod *corev1.Pod, nsLabels labels.Set) bool {
	if at.Namespaces.Has(pod.Namespace) || at.NamespaceSelector.Matches(nsLabels) {
		return at.Selector.Matches(labels.Set(pod.Labels))
	}
	return false
}

// WeightedAffinityTerm is a "processed" representation of v1.WeightedAffinityTerm.
type WeightedAffinityTerm struct {
	AffinityTerm
	Weight int32
}

// Diagnosis records the details to diagnose a scheduling failure.
type Diagnosis struct {
	NodeToStatusMap      NodeToStatusMap
	UnschedulablePlugins sets.String
	// PostFilterMsg records the messages returned from PostFilterPlugins.
	PostFilterMsg string
}

// FitError describes a fit error of a pod.
type FitError struct {
	Pod         *corev1.Pod
	NumAllNodes int
	Diagnosis   Diagnosis
}

const (
	// NoNodeAvailableMsg is used to format message when no nodes available.
	NoNodeAvailableMsg = "0/%v nodes are available"
)

// Error returns detailed information of why the pod failed to fit on each node
func (f *FitError) Error() string {
	reasons := make(map[string]int)
	for _, status := range f.Diagnosis.NodeToStatusMap {
		for _, reason := range status.Reasons() {
			reasons[reason]++
		}
	}

	sortReasonsHistogram := func() []string {
		var reasonStrings []string
		for k, v := range reasons {
			reasonStrings = append(reasonStrings, fmt.Sprintf("%v %v", v, k))
		}
		sort.Strings(reasonStrings)
		return reasonStrings
	}
	reasonMsg := fmt.Sprintf(NoNodeAvailableMsg+": %v.", f.NumAllNodes, strings.Join(sortReasonsHistogram(), ", "))
	postFilterMsg := f.Diagnosis.PostFilterMsg
	if postFilterMsg != "" {
		reasonMsg += " " + postFilterMsg
	}
	return reasonMsg
}

func newAffinityTerm(pod *corev1.Pod, term *corev1.PodAffinityTerm) (*AffinityTerm, error) {
	selector, err := metav1.LabelSelectorAsSelector(term.LabelSelector)
	if err != nil {
		return nil, err
	}

	namespaces := getNamespacesFromPodAffinityTerm(pod, term)
	nsSelector, err := metav1.LabelSelectorAsSelector(term.NamespaceSelector)
	if err != nil {
		return nil, err
	}

	return &AffinityTerm{Namespaces: namespaces, Selector: selector, TopologyKey: term.TopologyKey, NamespaceSelector: nsSelector}, nil
}

// getAffinityTerms receives a Pod and affinity terms and returns the namespaces and
// selectors of the terms.
func getAffinityTerms(pod *corev1.Pod, v1Terms []corev1.PodAffinityTerm) ([]AffinityTerm, error) {
	if v1Terms == nil {
		return nil, nil
	}

	var terms []AffinityTerm
	for i := range v1Terms {
		t, err := newAffinityTerm(pod, &v1Terms[i])
		if err != nil {
			// We get here if the label selector failed to process
			return nil, err
		}
		terms = append(terms, *t)
	}
	return terms, nil
}

// getWeightedAffinityTerms returns the list of processed affinity terms.
func getWeightedAffinityTerms(pod *corev1.Pod, v1Terms []corev1.WeightedPodAffinityTerm) ([]WeightedAffinityTerm, error) {
	if v1Terms == nil {
		return nil, nil
	}

	var terms []WeightedAffinityTerm
	for i := range v1Terms {
		t, err := newAffinityTerm(pod, &v1Terms[i].PodAffinityTerm)
		if err != nil {
			// We get here if the label selector failed to process
			return nil, err
		}
		terms = append(terms, WeightedAffinityTerm{AffinityTerm: *t, Weight: v1Terms[i].Weight})
	}
	return terms, nil
}

// NewPodInfo returns a new PodInfo.
func NewPodInfo(pod *corev1.Pod) *PodInfo {
	pInfo := &PodInfo{}
	pInfo.Update(pod)
	return pInfo
}

func getPodAffinityTerms(affinity *corev1.Affinity) (terms []corev1.PodAffinityTerm) {
	if affinity != nil && affinity.PodAffinity != nil {
		if len(affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) != 0 {
			terms = affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		}
		// TODO: Uncomment this block when implement RequiredDuringSchedulingRequiredDuringExecution.
		//if len(affinity.PodAffinity.RequiredDuringSchedulingRequiredDuringExecution) != 0 {
		//	terms = append(terms, affinity.PodAffinity.RequiredDuringSchedulingRequiredDuringExecution...)
		//}
	}
	return terms
}

func getPodAntiAffinityTerms(affinity *corev1.Affinity) (terms []corev1.PodAffinityTerm) {
	if affinity != nil && affinity.PodAntiAffinity != nil {
		if len(affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) != 0 {
			terms = affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		}
		// TODO: Uncomment this block when implement RequiredDuringSchedulingRequiredDuringExecution.
		//if len(affinity.PodAntiAffinity.RequiredDuringSchedulingRequiredDuringExecution) != 0 {
		//	terms = append(terms, affinity.PodAntiAffinity.RequiredDuringSchedulingRequiredDuringExecution...)
		//}
	}
	return terms
}

// returns a set of names according to the namespaces indicated in podAffinityTerm.
// If namespaces is empty it considers the given pod's namespace.
func getNamespacesFromPodAffinityTerm(pod *corev1.Pod, podAffinityTerm *corev1.PodAffinityTerm) sets.String {
	names := sets.String{}
	if len(podAffinityTerm.Namespaces) == 0 && podAffinityTerm.NamespaceSelector == nil {
		names.Insert(pod.Namespace)
	} else {
		names.Insert(podAffinityTerm.Namespaces...)
	}
	return names
}

// ImageStateSummary provides summarized information about the state of an image.
type ImageStateSummary struct {
	Size     int64 // Size of the image
	NumNodes int   // Used to track how many nodes have this image
}

// NodeInfo is node level aggregated information.
type NodeInfo struct {
	node                         *corev1.Node
	Pods                         map[string]*PodInfo
	PodsWithAffinity             []*PodInfo
	PodsWithRequiredAntiAffinity []*PodInfo
	UsedPorts                    HostPortInfo
	ImageStates                  map[string]*ImageStateSummary
	PVCRefCounts                 map[string]int
	GRPCHost                     string
	NodeMetric                   *NodeMetric
	GPUMetrics                   map[string]*GPUMetric
	IsGPUNode                    bool
	TotalGPUCount                int64
	PluginResult                 *PluginResult
	Requested                    *Resource
	Allocatable                  *Resource
	ReservePodList               sets.String
	// NonZeroRequested             *Resource
}

// Node returns overall information about this node.
func (n *NodeInfo) Node() *corev1.Node {
	if n == nil {
		return nil
	}
	return n.node
}

func NewNodeInfo() *NodeInfo {
	return &NodeInfo{
		node:                         nil,
		Pods:                         make(map[string]*PodInfo),
		PodsWithAffinity:             nil,
		PodsWithRequiredAntiAffinity: nil,
		UsedPorts:                    make(HostPortInfo),
		ImageStates:                  make(map[string]*ImageStateSummary),
		PVCRefCounts:                 make(map[string]int),
		GRPCHost:                     "",
		NodeMetric:                   NewNodeMetric(),
		GPUMetrics:                   make(map[string]*GPUMetric),
		IsGPUNode:                    true,
		TotalGPUCount:                0,
		PluginResult:                 NewPluginResult(),
		Requested:                    &Resource{},
		Allocatable:                  &Resource{},
		ReservePodList:               sets.NewString(),
		// NonZeroRequested:             &Resource{},
	}
}

func (n *NodeInfo) DumpNodeInfo() {
	fmt.Println("Dump Cache")

	fmt.Println("(1') Node() name {", n.Node().Name, "}")

	fmt.Print("(2) pods: ")
	for _, pod := range n.Pods {
		fmt.Print(pod.Pod.Name, ", ")
	}
	fmt.Println()

	// fmt.Print("(3) image: ")
	// for imageName, _ := range nodeInfo.ImageStates {
	// 	fmt.Print(imageName, ", ")
	// }
	// fmt.Println()

	// fmt.Print("(3) num of image: ", len(n.ImageStates), "\n")

	fmt.Print("(3) gpu name: ")
	for _, uuid := range n.NodeMetric.GPU_UUID {
		fmt.Print(uuid, ", ")
	}
	fmt.Println()

	fmt.Print("(4) gpu uuid: ")
	for uuid, _ := range n.GPUMetrics {
		fmt.Print(uuid, ", ")
	}
	fmt.Println()

}

// SetNode sets the overall node information.
func (n *NodeInfo) SetNode(node *corev1.Node) {
	n.node = node
	n.Allocatable = NewResource(node.Status.Allocatable)
}

func (n *NodeInfo) InitNodeInfo(node *corev1.Node, hostKubeClient *kubernetes.Clientset) error {
	//grpchost
	ip := GetMetricCollectorIP(n.Pods)
	n.GRPCHost = ip

	//pluginresult
	n.PluginResult = NewPluginResult()

	//init node,gpu metric
	err := n.GetInitMetric(ip)
	if err != nil {
		fmt.Println("get node {", node.Name, "} metric error!")
		n.PluginResult.IsFiltered = true
		return err
	}

	if isNonGPUNode(node) {
		n.IsGPUNode = false
	}

	return nil
}

//return whether the node is GPUNode or not
func isNonGPUNode(node *corev1.Node) bool {
	if _, ok := node.Labels["gpu"]; ok {
		return false
	}
	return true
}

// NewResource creates a Resource from ResourceList
func NewResource(rl v1.ResourceList) *Resource {
	r := &Resource{}
	r.Add(rl)
	return r
}

// Add adds ResourceList into Resource.
func (r *Resource) Add(rl v1.ResourceList) {
	if r == nil {
		return
	}

	for rName, rQuant := range rl {
		switch rName {
		case v1.ResourceCPU:
			r.MilliCPU += rQuant.MilliValue()
		case v1.ResourceMemory:
			r.Memory += rQuant.Value()
		case v1.ResourcePods:
			r.AllowedPodNumber += int(rQuant.Value())
		case v1.ResourceEphemeralStorage:
			// if utilfeature.DefaultFeatureGate.Enabled(features.LocalStorageCapacityIsolation) {
			// if the local storage capacity isolation feature gate is disabled, pods request 0 disk.
			r.EphemeralStorage += rQuant.Value()
			// }
			// default:
			// 	if schedutil.IsScalarResourceName(rName) {
			// 		r.AddScalar(rName, rQuant.Value())
			// 	}
		}
	}
}

func GetMetricCollectorIP(pods map[string]*PodInfo) string {
	for podName, pod := range pods {
		if strings.HasPrefix(podName, "keti-gpu-metric-collector") {
			return pod.Pod.Status.PodIP
		}
	}
	return ""
}

type NVLink struct {
	GPU1       string
	GPU2       string
	Link       int32
	Score      int
	IsSelected bool
	IsFiltered bool
}

func NewNVLink(s1 string, s2 string, l int32) NVLink {
	return NVLink{
		GPU1:       s1,
		GPU2:       s2,
		Link:       l,
		Score:      0,
		IsSelected: false,
		IsFiltered: false,
	}
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
	NVLinkList    []NVLink
}

func NewNodeMetric() *NodeMetric {
	return &NodeMetric{
		MilliCPUTotal: 0,
		MilliCPUUsed:  0,
		MemoryTotal:   0,
		MemoryUsed:    0,
		StorageTotal:  0,
		StorageUsed:   0,
		TotalGPUCount: 0,
		GPU_UUID:      nil,
		MaxGPUMemory:  0,
		NVLinkList:    nil,
	}
}

// each GPU metric
type GPUMetric struct {
	GPUName        string
	GPUIndex       int64
	GPUPowerUsed   int64
	GPUPowerTotal  int64
	GPUMemoryTotal int64
	GPUMemoryFree  int64
	GPUMemoryUsed  int64
	GPUTemperature int64
	PodCount       int64
	GPUFlops       int64
	GPUArch        int64
	GPUUtil        int64
}

func NewGPUMetric() *GPUMetric {
	return &GPUMetric{
		GPUName:        "",
		GPUIndex:       0,
		GPUPowerUsed:   0,
		GPUPowerTotal:  0,
		GPUMemoryTotal: 0,
		GPUMemoryFree:  0,
		GPUMemoryUsed:  0,
		GPUTemperature: 0,
		PodCount:       0,
		GPUFlops:       0,
		GPUArch:        0,
		GPUUtil:        0,
	}
}

// Resource is a collection of compute resource.
type Resource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
	AllowedPodNumber int
	ScalarResources  map[v1.ResourceName]int64
}

type PodResource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
	GPUCount         int
	GPUMemoryLimit   int64
	GPUMemoryRequest int64
}

// // Clone returns a copy of this resource.
// func (r *Resource) Clone() *Resource {
// 	res := &Resource{
// 		MilliCPU:         r.MilliCPU,
// 		Memory:           r.Memory,
// 		AllowedPodNumber: r.AllowedPodNumber,
// 		EphemeralStorage: r.EphemeralStorage,
// 	}
// 	if r.ScalarResources != nil {
// 		res.ScalarResources = make(map[v1.ResourceName]int64)
// 		for k, v := range r.ScalarResources {
// 			res.ScalarResources[k] = v
// 		}
// 	}
// 	return res
// }

// // AddScalar adds a resource by a scalar value of this resource.
// func (r *Resource) AddScalar(name v1.ResourceName, quantity int64) {
// 	r.SetScalar(name, r.ScalarResources[name]+quantity)
// }

// // SetScalar sets a resource by a scalar value of this resource.
// func (r *Resource) SetScalar(name v1.ResourceName, quantity int64) {
// 	// Lazily allocate scalar resource map.
// 	if r.ScalarResources == nil {
// 		r.ScalarResources = map[v1.ResourceName]int64{}
// 	}
// 	r.ScalarResources[name] = quantity
// }

// // // SetMaxResource compares with ResourceList and takes max value for each Resource.
// // func (r *Resource) SetMaxResource(rl v1.ResourceList) {
// // 	if r == nil {
// // 		return
// // 	}

// // 	for rName, rQuantity := range rl {
// // 		switch rName {
// // 		case v1.ResourceMemory:
// // 			r.Memory = max(r.Memory, rQuantity.Value())
// // 		case v1.ResourceCPU:
// // 			r.MilliCPU = max(r.MilliCPU, rQuantity.MilliValue())
// // 		case v1.ResourceEphemeralStorage:
// // 			if utilfeature.DefaultFeatureGate.Enabled(features.LocalStorageCapacityIsolation) {
// // 				r.EphemeralStorage = max(r.EphemeralStorage, rQuantity.Value())
// // 			}
// // 		default:
// // 			if schedutil.IsScalarResourceName(rName) {
// // 				r.SetScalar(rName, max(r.ScalarResources[rName], rQuantity.Value()))
// // 			}
// // 		}
// // 	}
// // }

// // Node returns overall information about this node.
// func (n *NodeInfo) Node() *corev1.Node {
// 	if n == nil {
// 		return nil
// 	}
// 	return n.node
// }

// // // Clone returns a copy of this node.
// // func (n *NodeInfo) Clone() *NodeInfo {
// // 	clone := &NodeInfo{
// // 		node:      n.node,
// // 		Requested: n.Requested.Clone(),
// // 		// NonZeroRequested: n.NonZeroRequested.Clone(),
// // 		Allocatable:  n.Allocatable.Clone(),
// // 		UsedPorts:    make(HostPortInfo),
// // 		ImageStates:  n.ImageStates,
// // 		PVCRefCounts: n.PVCRefCounts,
// // 		Generation:   n.Generation,
// // 	}
// // 	if len(n.Pods) > 0 {
// // 		clone.Pods = append([]*PodInfo(nil), n.Pods...)
// // 	}
// // 	if len(n.UsedPorts) > 0 {
// // 		// HostPortInfo is a map-in-map struct
// // 		// make sure it's deep copied
// // 		for ip, portMap := range n.UsedPorts {
// // 			clone.UsedPorts[ip] = make(map[ProtocolPort]struct{})
// // 			for protocolPort, v := range portMap {
// // 				clone.UsedPorts[ip][protocolPort] = v
// // 			}
// // 		}
// // 	}
// // 	if len(n.PodsWithAffinity) > 0 {
// // 		clone.PodsWithAffinity = append([]*PodInfo(nil), n.PodsWithAffinity...)
// // 	}
// // 	if len(n.PodsWithRequiredAntiAffinity) > 0 {
// // 		clone.PodsWithRequiredAntiAffinity = append([]*PodInfo(nil), n.PodsWithRequiredAntiAffinity...)
// // 	}
// // 	return clone
// // }

// // String returns representation of human readable format of this NodeInfo.
// func (n *NodeInfo) String() string {
// 	podKeys := make([]string, len(n.Pods))
// 	for i, p := range n.Pods {
// 		podKeys[i] = p.Pod.Name
// 	}
// 	return fmt.Sprintf("&NodeInfo{Pods:%v, RequestedResource:%#v, UsedPort: %#v, AllocatableResource:%#v}",
// 		podKeys, n.Requested, n.UsedPorts, n.Allocatable)
// }

// AddPodInfo adds pod information to this NodeInfo.
// Consider using this instead of AddPod if a PodInfo is already computed.
func (n *NodeInfo) AddPodInfo(podInfo *PodInfo) {
	res, _ := calculatePodResource(podInfo.Pod)
	n.Requested.MilliCPU += res.MilliCPU
	n.Requested.Memory += res.Memory
	n.Requested.EphemeralStorage += res.EphemeralStorage
	// if n.Requested.ScalarResources == nil && len(res.ScalarResources) > 0 {
	// 	n.Requested.ScalarResources = map[v1.ResourceName]int64{}
	// }
	// for rName, rQuant := range res.ScalarResources {
	// 	n.Requested.ScalarResources[rName] += rQuant
	// }
	n.Pods[podInfo.Pod.Name] = podInfo
	if podWithAffinity(podInfo.Pod) {
		n.PodsWithAffinity = append(n.PodsWithAffinity, podInfo)
	}
	if podWithRequiredAntiAffinity(podInfo.Pod) {
		n.PodsWithRequiredAntiAffinity = append(n.PodsWithRequiredAntiAffinity, podInfo)
	}

	// Consume ports when pods added.
	n.updateUsedPorts(podInfo.Pod, true)
	n.updatePVCRefCounts(podInfo.Pod, true)
}

// AddPod is a wrapper around AddPodInfo.
func (n *NodeInfo) AddPod(pod corev1.Pod) {
	n.AddPodInfo(NewPodInfo(&pod))
}

func podWithAffinity(p *corev1.Pod) bool {
	affinity := p.Spec.Affinity
	return affinity != nil && (affinity.PodAffinity != nil || affinity.PodAntiAffinity != nil)
}

func podWithRequiredAntiAffinity(p *corev1.Pod) bool {
	affinity := p.Spec.Affinity
	return affinity != nil && affinity.PodAntiAffinity != nil &&
		len(affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) != 0
}

func removeFromSlice(s []*PodInfo, k string) []*PodInfo {
	for i := range s {
		k2, err := GetPodKey(s[i].Pod)
		if err != nil {
			klog.ErrorS(err, "Cannot get pod key", "pod", klog.KObj(s[i].Pod))
			continue
		}
		if k == k2 {
			// delete the element
			s[i] = s[len(s)-1]
			s = s[:len(s)-1]
			break
		}
	}
	return s
}

// RemovePod subtracts pod information from this NodeInfo.
func (n *NodeInfo) RemovePod(pod *corev1.Pod) error {
	k, err := GetPodKey(pod)
	if err != nil {
		return err
	}
	if podWithAffinity(pod) {
		n.PodsWithAffinity = removeFromSlice(n.PodsWithAffinity, k)
	}
	if podWithRequiredAntiAffinity(pod) {
		n.PodsWithRequiredAntiAffinity = removeFromSlice(n.PodsWithRequiredAntiAffinity, k)
	}

	// for i := range n.Pods {
	// 	k2, err := GetPodKey(n.Pods[i].Pod)
	// 	if err != nil {
	// 		klog.ErrorS(err, "Cannot get pod key", "pod", klog.KObj(n.Pods[i].Pod))
	// 		continue
	// 	}
	// 	if k == k2 {
	// delete the element
	delete(n.Pods, pod.Name)
	// n.Pods[i] = n.Pods[len(n.Pods)-1]
	// n.Pods = n.Pods[:len(n.Pods)-1]
	// reduce the resource data
	res, _ := calculatePodResource(pod)

	n.Requested.MilliCPU -= res.MilliCPU
	n.Requested.Memory -= res.Memory
	n.Requested.EphemeralStorage -= res.EphemeralStorage
	// if len(res.ScalarResources) > 0 && n.Requested.ScalarResources == nil {
	// 	n.Requested.ScalarResources = map[v1.ResourceName]int64{}
	// }
	// for rName, rQuant := range res.ScalarResources {
	// 	n.Requested.ScalarResources[rName] -= rQuant
	// }
	// n.NonZeroRequested.MilliCPU -= non0CPU
	// n.NonZeroRequested.Memory -= non0Mem

	// Release ports when remove Pods.
	n.updateUsedPorts(pod, false)
	n.updatePVCRefCounts(pod, false)
	n.resetSlicesIfEmpty()
	return nil
	// }
	// }
	// return fmt.Errorf("no corresponding pod %s in pods of node %s", pod.Name, n.node.Name)
}

// resets the slices to nil so that we can do DeepEqual in unit tests.
func (n *NodeInfo) resetSlicesIfEmpty() {
	if len(n.PodsWithAffinity) == 0 {
		n.PodsWithAffinity = nil
	}
	if len(n.PodsWithRequiredAntiAffinity) == 0 {
		n.PodsWithRequiredAntiAffinity = nil
	}
	if len(n.Pods) == 0 {
		n.Pods = nil
	}
}

func Max(a, b int64) int64 {
	if a >= b {
		return a
	}
	return b
}

func calculatePodResource(pod *corev1.Pod) (*PodResource, bool) {
	res := &PodResource{0, 0, 0, 0, 0, 0}
	isGPUPod := false

	//resource: gpucount, cpu, memory, storage, gpumemory
	for _, c := range pod.Spec.Containers {
		GPUMPSLimit := c.Resources.Limits["keti.com/mpsgpu"]
		if GPUMPSLimit.String() != "" {
			gc, _ := strconv.Atoi(GPUMPSLimit.String())
			res.GPUCount += gc
			isGPUPod = true
		}
		for rName, rQuant := range c.Resources.Requests {
			switch rName {
			case corev1.ResourceCPU:
				res.MilliCPU += int64(rQuant.MilliValue())
			case corev1.ResourceMemory:
				res.Memory += int64(rQuant.Value())
			case corev1.ResourceEphemeralStorage:
				res.EphemeralStorage += int64(rQuant.Value())
			}
		}
		for rName, rQuant := range c.Resources.Limits {
			switch rName {
			case corev1.ResourceCPU:
				if res.MilliCPU == 0 {
					res.MilliCPU += int64(rQuant.MilliValue())
				}
			case corev1.ResourceMemory:
				if res.Memory == 0 {
					res.Memory += int64(rQuant.MilliValue())
				}
			}
		}
	}

	//resource: gpucount, cpu, memory, storage, gpumemory
	for _, ic := range pod.Spec.InitContainers {
		GPUMPSLimit := ic.Resources.Limits["keti.com/mpsgpu"]
		if GPUMPSLimit.String() != "" {
			gc, _ := strconv.Atoi(GPUMPSLimit.String())
			res.GPUCount += gc
			isGPUPod = true
		}
		for rName, rQuant := range ic.Resources.Requests {
			switch rName {
			case corev1.ResourceCPU:
				res.MilliCPU += int64(rQuant.MilliValue())
			case corev1.ResourceMemory:
				res.Memory += int64(rQuant.Value())
			case corev1.ResourceEphemeralStorage:
				res.EphemeralStorage += int64(rQuant.Value())
			}
		}
		for rName, rQuant := range ic.Resources.Limits {
			switch rName {
			case corev1.ResourceCPU:
				if res.MilliCPU == 0 {
					res.MilliCPU += int64(rQuant.MilliValue())
				}
			case corev1.ResourceMemory:
				if res.Memory == 0 {
					res.Memory += int64(rQuant.MilliValue())
				}
			}
		}
	}

	//annotation: GPUlimit, GPURequest
	limit := pod.ObjectMeta.Annotations["GPUlimits"]
	request := pod.ObjectMeta.Annotations["GPUrequest"]
	if request == "" && limit != "" {
		res.GPUMemoryRequest = getMemory(limit)
		res.GPUMemoryLimit = getMemory(limit)
	} else {
		res.GPUMemoryRequest = getMemory(request)
		res.GPUMemoryLimit = getMemory(limit)
	}

	return res, isGPUPod
}

// updateUsedPorts updates the UsedPorts of NodeInfo.
func (n *NodeInfo) updateUsedPorts(pod *corev1.Pod, add bool) {
	for _, container := range pod.Spec.Containers {
		for _, podPort := range container.Ports {
			if add {
				n.UsedPorts.Add(podPort.HostIP, string(podPort.Protocol), podPort.HostPort)
			} else {
				n.UsedPorts.Remove(podPort.HostIP, string(podPort.Protocol), podPort.HostPort)
			}
		}
	}
}

// updatePVCRefCounts updates the PVCRefCounts of NodeInfo.
func (n *NodeInfo) updatePVCRefCounts(pod *corev1.Pod, add bool) {
	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim == nil {
			continue
		}

		key := pod.Namespace + "/" + v.PersistentVolumeClaim.ClaimName
		if add {
			n.PVCRefCounts[key] += 1
		} else {
			n.PVCRefCounts[key] -= 1
			if n.PVCRefCounts[key] <= 0 {
				delete(n.PVCRefCounts, key)
			}
		}
	}
}

// // SetNode sets the overall node information.
// func (n *NodeInfo) SetNode(node *corev1.Node) {
// 	n.node = node
// 	n.Allocatable = NewNodeResource(node.Status.Allocatable)
// 	n.Generation = nextGeneration()
// }

// RemoveNode removes the node object, leaving all other tracking information.
func (n *NodeInfo) RemoveNode() {
	n.node = nil
}

// // FilterOutPods receives a list of pods and filters out those whose node names
// // are equal to the node of this NodeInfo, but are not found in the pods of this NodeInfo.
// //
// // Preemption logic simulates removal of pods on a node by removing them from the
// // corresponding NodeInfo. In order for the simulation to work, we call this method
// // on the pods returned from SchedulerCache, so that predicate functions see
// // only the pods that are not removed from the NodeInfo.
// func (n *NodeInfo) FilterOutPods(pods []*corev1.Pod) []*corev1.Pod {
// 	node := n.Node()
// 	if node == nil {
// 		return pods
// 	}
// 	filtered := make([]*corev1.Pod, 0, len(pods))
// 	for _, p := range pods {
// 		if p.Spec.NodeName != node.Name {
// 			filtered = append(filtered, p)
// 			continue
// 		}
// 		// If pod is on the given node, add it to 'filtered' only if it is present in nodeInfo.
// 		podKey, err := GetPodKey(p)
// 		if err != nil {
// 			continue
// 		}
// 		for _, np := range n.Pods {
// 			npodkey, _ := GetPodKey(np.Pod)
// 			if npodkey == podKey {
// 				filtered = append(filtered, p)
// 				break
// 			}
// 		}
// 	}
// 	return filtered
// }

// GetPodKey returns the string key of a pod.
func GetPodKey(pod *corev1.Pod) (string, error) {
	uid := string(pod.UID)
	if len(uid) == 0 {
		return "", errors.New("cannot get cache key for pod with empty UID")
	}
	return uid, nil
}

// DefaultBindAllHostIP defines the default ip address used to bind to all host.
const DefaultBindAllHostIP = "0.0.0.0"

// ProtocolPort represents a protocol port pair, e.g. tcp:80.
type ProtocolPort struct {
	Protocol string
	Port     int32
}

// NewProtocolPort creates a ProtocolPort instance.
func NewProtocolPort(protocol string, port int32) *ProtocolPort {
	pp := &ProtocolPort{
		Protocol: protocol,
		Port:     port,
	}

	if len(pp.Protocol) == 0 {
		pp.Protocol = string(v1.ProtocolTCP)
	}

	return pp
}

// HostPortInfo stores mapping from ip to a set of ProtocolPort
type HostPortInfo map[string]map[ProtocolPort]struct{}

// Add adds (ip, protocol, port) to HostPortInfo
func (h HostPortInfo) Add(ip, protocol string, port int32) {
	if port <= 0 {
		return
	}

	h.sanitize(&ip, &protocol)

	pp := NewProtocolPort(protocol, port)
	if _, ok := h[ip]; !ok {
		h[ip] = map[ProtocolPort]struct{}{
			*pp: {},
		}
		return
	}

	h[ip][*pp] = struct{}{}
}

// Remove removes (ip, protocol, port) from HostPortInfo
func (h HostPortInfo) Remove(ip, protocol string, port int32) {
	if port <= 0 {
		return
	}

	h.sanitize(&ip, &protocol)

	pp := NewProtocolPort(protocol, port)
	if m, ok := h[ip]; ok {
		delete(m, *pp)
		if len(h[ip]) == 0 {
			delete(h, ip)
		}
	}
}

// Len returns the total number of (ip, protocol, port) tuple in HostPortInfo
func (h HostPortInfo) Len() int {
	length := 0
	for _, m := range h {
		length += len(m)
	}
	return length
}

// CheckConflict checks if the input (ip, protocol, port) conflicts with the existing
// ones in HostPortInfo.
func (h HostPortInfo) CheckConflict(ip, protocol string, port int32) bool {
	if port <= 0 {
		return false
	}

	h.sanitize(&ip, &protocol)

	pp := NewProtocolPort(protocol, port)

	// If ip is 0.0.0.0 check all IP's (protocol, port) pair
	if ip == DefaultBindAllHostIP {
		for _, m := range h {
			if _, ok := m[*pp]; ok {
				return true
			}
		}
		return false
	}

	// If ip isn't 0.0.0.0, only check IP and 0.0.0.0's (protocol, port) pair
	for _, key := range []string{DefaultBindAllHostIP, ip} {
		if m, ok := h[key]; ok {
			if _, ok2 := m[*pp]; ok2 {
				return true
			}
		}
	}

	return false
}

// sanitize the parameters
func (h HostPortInfo) sanitize(ip, protocol *string) {
	if len(*ip) == 0 {
		*ip = DefaultBindAllHostIP
	}
	if len(*protocol) == 0 {
		*protocol = string(v1.ProtocolTCP)
	}
}
