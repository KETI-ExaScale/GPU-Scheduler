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
	"fmt"
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

type Code int

const (
	Success                      Code = iota //not filtered
	Wait                                     //pending
	Error                                    //error
	Unschedulable                            //need rescheduling or wait
	UnschedulableAndUnresolvable             //cannot scheduling
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
	FilteredStatus    FilterStatus
	NodeScore         int
	GPUScores         map[string]*GPUScore
	TotalGPUScore     int
	TotalScore        int
	BestGPU           string
}

type GPUScore struct {
	UUID           string
	IsFiltered     bool
	FilteredStatus FilterStatus
	GPUScore       int
	IsSelected     bool
}

type FilterStatus struct {
	Code   Code //Unschedulable, UnschedulableAndUnresolvable, Error
	Stage  string
	Reason string
	Error  error
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
		FilteredStatus:    NewFilterStatus(),
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
	pr.FilteredStatus = NewFilterStatus()
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
		UUID:           uuid,
		IsFiltered:     false,
		FilteredStatus: NewFilterStatus(),
		GPUScore:       MinScore,
		IsSelected:     false,
	}
}

func (gs *GPUScore) InitGPUScore(uuid string) {
	gs.UUID = uuid
	gs.GPUScore = MinScore
	gs.FilteredStatus = NewFilterStatus()
	gs.IsFiltered = false
	gs.IsSelected = false
}

func NewFilterStatus() FilterStatus {
	return FilterStatus{
		Code:   Success,
		Stage:  "",
		Reason: "",
		Error:  nil,
	}
}

func (pr *PluginResult) FilterNode(node string, status FilterStatus) {
	KETI_LOG_L1(fmt.Sprintf("# node {%s} filtered, stage = %s", node, status.Stage))
	pr.IsFiltered = true
	pr.FilteredStatus = status
}

func (gs *GPUScore) FilterGPU(node string, gpu string, status FilterStatus) {
	KETI_LOG_L1(fmt.Sprintf("# node {%s} - gpu {%s} filtered, stage = %s", node, gpu, status.Stage))
	gs.IsFiltered = true
	gs.FilteredStatus = status
}

type QueuedPodInfo struct {
	*PodInfo
	PodUID                  types.UID
	Timestamp               time.Time //해당 큐에 들어온 시간
	Attempts                int       //스케줄링 시도 횟수
	InitialAttemptTimestamp time.Time //큐에 최초로 들어온 시간 (생성시간)
	Status                  *Status   //파드 스케줄링 상태
	TargetCluster           string    //생성 클러스터
	FilteredCluster         []string  //클러스터 매니저가 필터링할 클러스터
	PriorityScore           int       //우선순위큐 스코어
	UserPriority            int       //사용자 설정 우선순위
}

func NewQueuedPodInfo(pod *corev1.Pod) *QueuedPodInfo {
	if pod == nil { //Cluster Manager Init Pod
		return &QueuedPodInfo{
			PodUID:                  "",
			PodInfo:                 NewInitPodInfo(),
			Timestamp:               time.Now(),
			Attempts:                0,
			InitialAttemptTimestamp: time.Now(),
			Status:                  NewStatus(),
			TargetCluster:           "",
			FilteredCluster:         nil,
			PriorityScore:           0,
			UserPriority:            MiddlePriority,
		}
	}

	priority := regularPriority(pod.Annotations["priority"])
	targetCluster := pod.Annotations["clusterName"]

	return &QueuedPodInfo{ // Schedule Pod
		PodUID:                  pod.UID,
		PodInfo:                 GetNewPodInfo(pod),
		Timestamp:               time.Now(),
		Attempts:                0,
		InitialAttemptTimestamp: time.Now(),
		Status:                  NewStatus(),
		TargetCluster:           targetCluster,
		FilteredCluster:         nil,
		PriorityScore:           priority,
		UserPriority:            priority,
	}
}

func regularPriority(p string) int {
	p = strings.ToUpper(p)
	if p == "L" || p == "LOW" {
		return LowPriority
	} else if p == "H" || p == "HIGH" {
		return HighPriority
	} else if p == "I" || p == "Immediatly" {
		return Immediatly
	} else { //middle
		return MiddlePriority
	}
}

func (qpi *QueuedPodInfo) DeepCopy() *QueuedPodInfo {
	return &QueuedPodInfo{
		PodUID:                  qpi.PodUID,
		PodInfo:                 qpi.PodInfo.DeepCopy(),
		Timestamp:               qpi.Timestamp,
		Attempts:                qpi.Attempts,
		InitialAttemptTimestamp: qpi.InitialAttemptTimestamp,
		Status:                  qpi.Status,
		TargetCluster:           qpi.TargetCluster,
	}
}

type PodInfo struct {
	Pod                        *corev1.Pod
	RequiredAffinityTerms      []AffinityTerm
	RequiredAntiAffinityTerms  []AffinityTerm
	PreferredAffinityTerms     []WeightedAffinityTerm
	PreferredAntiAffinityTerms []WeightedAffinityTerm
	PodVolumesByNode           map[string]*PodVolumes
	ParseError                 error
	RequestedResource          *PodResource
	IsGPUPod                   bool
	ReserveNode                string
}

type PodVolumes struct {
	// StaticBindings are binding decisions for PVCs which can be bound to
	// pre-provisioned static PVs.
	StaticBindings []*BindingInfo
	// DynamicProvisions are PVCs that require dynamic provisioning
	DynamicProvisions []*v1.PersistentVolumeClaim
}

// BindingInfo holds a binding between PV and PVC.
type BindingInfo struct {
	// PVC that needs to be bound
	Pvc *v1.PersistentVolumeClaim

	// Proposed PV to bind to this PVC
	Pv *v1.PersistentVolume
}

func NewInitPodInfo() *PodInfo {
	cpu := "500m"
	memory := "1Gi"
	cpuQuentity := resource.MustParse(cpu)
	cpuMillivalue := int64(cpuQuentity.MilliValue())
	memoryQuentity := resource.MustParse(memory)
	memoryValue := int64(memoryQuentity.Value())
	res := &PodResource{cpuMillivalue, memoryValue, 0, 1, 0, 0}
	// &PodResource{MilliCPU Memory EphemeralStorage GPUCount GPUMemoryLimit GPUMemoryRequest}
	KETI_LOG_L1(fmt.Sprintf("<test> res: %+v", res))
	return &PodInfo{
		Pod:                        nil,
		RequiredAffinityTerms:      nil,
		RequiredAntiAffinityTerms:  nil,
		PreferredAffinityTerms:     nil,
		PreferredAntiAffinityTerms: nil,
		ParseError:                 nil,
		RequestedResource:          res,
		IsGPUPod:                   true,
		ReserveNode:                "",
	}
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

// // Diagnosis records the details to diagnose a scheduling failure.
// type Diagnosis struct {
// 	NodeToStatusMap      NodeToStatusMap
// 	UnschedulablePlugins sets.String
// 	// PostFilterMsg records the messages returned from PostFilterPlugins.
// 	PostFilterMsg string
// }

// // FitError describes a fit error of a pod.
// type FitError struct {
// 	Pod         *corev1.Pod
// 	NumAllNodes int
// 	Diagnosis   Diagnosis
// }

// const (
// 	// NoNodeAvailableMsg is used to format message when no nodes available.
// 	NoNodeAvailableMsg = "0/%v nodes are available"
// )

// // Error returns detailed information of why the pod failed to fit on each node
// func (f *FitError) Error() string {
// 	reasons := make(map[string]int)
// 	for _, status := range f.Diagnosis.NodeToStatusMap {
// 		for _, reason := range status.Reasons() {
// 			reasons[reason]++
// 		}
// 	}

// 	sortReasonsHistogram := func() []string {
// 		var reasonStrings []string
// 		for k, v := range reasons {
// 			reasonStrings = append(reasonStrings, fmt.Sprintf("%v %v", v, k))
// 		}
// 		sort.Strings(reasonStrings)
// 		return reasonStrings
// 	}
// 	reasonMsg := fmt.Sprintf(NoNodeAvailableMsg+": %v.", f.NumAllNodes, strings.Join(sortReasonsHistogram(), ", "))
// 	postFilterMsg := f.Diagnosis.PostFilterMsg
// 	if postFilterMsg != "" {
// 		reasonMsg += " " + postFilterMsg
// 	}
// 	return reasonMsg
// }

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
	// Overall node information.
	node *corev1.Node
	// Pods running on the node.
	Pods []*PodInfo
	// The subset of pods with affinity.
	PodsWithAffinity []*PodInfo
	// The subset of pods with required anti-affinity.
	PodsWithRequiredAntiAffinity []*PodInfo
	// Ports allocated on the node.
	UsedPorts HostPortInfo
	// ImageStates holds the entry of an image if and only if this image is on the node. The entry can be used for
	// checking an image's existence and advanced usage (e.g., image locality scheduling policy) based on the image
	// state information.
	ImageStates map[string]*ImageStateSummary
	// PVCRefCounts contains a mapping of PVC names to the number of pods on the node using it.
	// Keys are in the format "namespace/name".
	PVCRefCounts map[string]int
	// node "keti-gpu-metric-collector" ip
	// MetricCollectorIP string
	// nodemetric from "keti-gpu-metric-collector"
	// NodeMetric *NodeMetric //수정
	NVLinkList []*NVLink
	// gpumetric from "keti-gpu-metric-collector"
	// GPUMetrics map[string]*GPUMetric
	GPU_UUID []string //GPU UUID
	// IsGPUNode bool //수정
	// gpu count
	TotalGPUCount int64
	// scheduling plugin result
	PluginResult *PluginResult
	// Total requested resources of all pods on this node. This includes assumed
	// pods, which scheduler has sent for binding, but may not be scheduled yet.
	Requested *Resource
	// We store allocatedResources (which is Node.Status.Allocatable.*) explicitly
	// as int64, to avoid conversions and accessing map.
	Allocatable *Resource
	// node reserved flag
	Reserved bool
	// ReservePodList sets.String // ??
	// Avaliable                    bool // PluginResult.IsFiltered로 판별가능 -> Metric Collector가 있는지 여부로 초기화시 결정
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
		Pods:                         nil,
		PodsWithAffinity:             nil,
		PodsWithRequiredAntiAffinity: nil,
		UsedPorts:                    make(HostPortInfo),
		ImageStates:                  make(map[string]*ImageStateSummary),
		PVCRefCounts:                 make(map[string]int),
		NVLinkList:                   make([]*NVLink, 0),
		GPU_UUID:                     make([]string, 0),
		TotalGPUCount:                0,
		PluginResult:                 NewPluginResult(),
		Requested:                    &Resource{},
		Allocatable:                  &Resource{},
		Reserved:                     false,
	}
}

// SetNode sets the overall node information.
func (n *NodeInfo) SetNode(node *corev1.Node) {
	n.node = node
	n.Allocatable = NewResource(node.Status.Allocatable)
}

func (n *NodeInfo) InitNodeInfo(node *corev1.Node, hostKubeClient *kubernetes.Clientset) {
	n.PluginResult = NewPluginResult() //pluginresult

	err := n.GetGPUInitInfo(node)
	if err != nil {
		KETI_LOG_L3(fmt.Sprintf("<error> get gpu init info error-%s", err))
	} //node의 host-api-service와 통신해서 gpu_uuid, count, nvlink init 획득
}

// return whether the node is GPUNode or not
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
			r.EphemeralStorage += rQuant.Value()
		}
	}
}

func GetMetricCollectorIP(pods []*PodInfo) string {
	for _, pod := range pods {
		if strings.HasPrefix(pod.Pod.Name, "keti-gpu-metric-collector") && pod.Pod.Status.Phase == "Running" {
			return pod.Pod.Status.PodIP
		}
	}
	return ""
}

type NVLink struct {
	GPU1       string
	GPU2       string
	Lane       int32
	Score      int
	IsSelected bool
	IsFiltered bool
}

func NewNVLink(s1 string, s2 string, l int32) NVLink {
	return NVLink{
		GPU1:       s1,
		GPU2:       s2,
		Lane:       l,
		Score:      0,
		IsSelected: false,
		IsFiltered: false,
	}
}

func (n *NodeInfo) InitNVLinkList() {
	for _, nvlink := range n.NVLinkList {
		nvlink.Score = 0
		nvlink.IsFiltered = false
		nvlink.IsSelected = false
	}
}

// func (n *NodeInfo) gpuPodCountDown() error {
// 	if n.GPUPodcount == 0 {
// 		return fmt.Errorf("gpu metric pod count = 0")
// 	}
// 	n.GPUPodcount--
// 	return nil
// }

// func (n *NodeInfo) gpuPodCountUp() {
// 	n.GPUPodcount++
// }

// Resource is a collection of compute resource.
type Resource struct {
	MilliCPU         int64                     // node cpu
	Memory           int64                     // node memory
	EphemeralStorage int64                     // node storage
	AllowedPodNumber int                       // node allowed pod number
	PodCount         int                       // node running pod number
	ScalarResources  map[v1.ResourceName]int64 // custom resources
}

type PodResource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
	GPUCount         int
	GPUMemoryLimit   int64
	GPUMemoryRequest int64
}

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
	n.Pods = append(n.Pods, podInfo)
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

	n.Pods = removeFromSlice(n.Pods, k)
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

// func (n *NodeInfo) gpuPodCountDown(pod *corev1.Pod) {
// 	uuids := pod.ObjectMeta.Annotations["UUID"]
// 	uuid_list := strings.Split(uuids, ",")

// 	for _, uuid := range uuid_list {
// 		n.GPUMetrics[uuid].gpuPodCountDown()
// 	}
// }

// func (n *NodeInfo) gpuPodCountUp(uuids string) {
// 	uuid_list := strings.Split(uuids, ",")

// 	for _, uuid := range uuid_list {
// 		n.GPUMetrics[uuid].gpuPodCountUp()
// 	}
// }

// GetPodKey returns the string key of a pod.
func GetPodKey(pod *corev1.Pod) (string, error) {
	// uid := string(pod.UID)
	// if len(uid) == 0 {
	// 	return "", errors.New("cannot get cache key for pod with empty UID")
	// }
	// return uid, nil
	pName := string(pod.Name)
	return pName, nil
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

// type Desc [2]string

type Status struct {
	Code    Code   //파드 스케줄링 상태
	Reasons string //이유
	Err     error  //에러내용
	// FilteredPlugin map[string]Desc //filter plugin) key: 노드네임 - value: [0]단계, [1]이유
}

func NewStatus() *Status {
	return &Status{
		Code:    Wait,
		Reasons: "",
		Err:     nil,
		// FilteredPlugin: make(map[string]Desc),
	}
}
