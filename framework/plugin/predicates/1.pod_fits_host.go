package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"

	v1 "k8s.io/api/core/v1"
)

type PodFitsHost struct{}

func (pl PodFitsHost) Name() string {
	return "PodFitsHost"
}

func (pl PodFitsHost) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#1. %s", pl.Name()))
}

func (pl PodFitsHost) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	if len(newPod.Pod.Spec.NodeName) == 0 {
		return
	}

	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if !Fits(newPod.Pod, nodeName) {
				reason := fmt.Sprintf("node name=%s not fit pod.spec.nodeName=%s", nodeName, newPod.Pod.Spec.NodeName)
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
			}
		}
	}
}

// Fits actually checks if the pod fits the node.
func Fits(pod *v1.Pod, nodeName string) bool {
	return len(pod.Spec.NodeName) == 0 || pod.Spec.NodeName == nodeName
}
