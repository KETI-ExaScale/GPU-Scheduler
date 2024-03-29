package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
	v1helper "k8s.io/component-helpers/scheduling/corev1"
)

type CheckNodeUnschedulable struct{}

func (pl CheckNodeUnschedulable) Name() string {
	return "CheckNodeUnschedulable"
}

func (pl CheckNodeUnschedulable) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("[stage] F#2. %s", pl.Name()))
}

func (pl CheckNodeUnschedulable) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			podToleratesUnschedulable := v1helper.TolerationsTolerateTaint(newPod.Pod.Spec.Tolerations, &corev1.Taint{
				Key:    corev1.TaintNodeUnschedulable,
				Effect: corev1.TaintEffectNoSchedule,
			})

			if nodeinfo.Node().Spec.Unschedulable && !podToleratesUnschedulable {
				reason := fmt.Sprintf("node unschedulable and pod not tolerates unschedulable")
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
			}
		}
	}
}
