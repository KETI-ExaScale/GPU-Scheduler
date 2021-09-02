package resourceinfo

import (
	corev1 "k8s.io/api/core/v1"
)

var AvailableNodeCount *int64 = new(int64)

type NodeInfo struct {
	// Overall node information.
	NodeName   string
	Node       corev1.Node
	Pods       []*corev1.Pod
	Affinity   map[string]string
	NodeScore  float64
	IsFiltered bool
}

type NodeMetric struct {
	//Overall node metric information.
	NodeName   string
	NodeCPU    string
	NodeMemory string
	GPUCount   int
	UUID       string
}

func (n *NodeInfo) FilterNode() error {
	n.IsFiltered = true
	*AvailableNodeCount--
	return nil
}

// Resource is a collection of compute resource.
type Resource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
}

type PodWatchEvent struct {
	Type   string     `json:"type"`
	Object corev1.Pod `json:"object"`
}

//return if the node is master or not
func IsMaster(node corev1.Node) bool {
	if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
		return true
	}

	return false
}
