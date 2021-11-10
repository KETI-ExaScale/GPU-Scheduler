package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	v1helper "k8s.io/component-helpers/scheduling/corev1"
)

func CheckNodeUnschedulable(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-2] Filtering > CheckNodeUnschedulable")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			// If pod tolerate unschedulable taint, it's also tolerate `node.Spec.Unschedulable`.
			podToleratesUnschedulable := v1helper.TolerationsTolerateTaint(newPod.Pod.Spec.Tolerations, &v1.Taint{
				Key:    corev1.TaintNodeUnschedulable,
				Effect: corev1.TaintEffectNoSchedule,
			})

			// TODO (k82cn): deprecates `node.Spec.Unschedulable` in 1.13.
			if nodeinfo.Node.Spec.Unschedulable && !podToleratesUnschedulable {
				nodeinfo.FilterNode()
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> check_node_unschedulable")
	}

	return nil
}
