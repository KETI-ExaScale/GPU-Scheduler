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

func CheckNodeUnschedulable(newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-2] Filtering > CheckNodeUnschedulable")
	}

	for _, nodeinfo := range resource.NodeInfoList {
		if !nodeinfo.IsFiltered {
			podToleratesUnschedulable := v1helper.TolerationsTolerateTaint(newPod.Pod.Spec.Tolerations, &v1.Taint{
				Key:    corev1.TaintNodeUnschedulable,
				Effect: corev1.TaintEffectNoSchedule,
			})

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
