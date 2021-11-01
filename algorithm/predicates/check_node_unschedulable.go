package predicates

import (
	"fmt"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"
	"log"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	v1helper "k8s.io/component-helpers/scheduling/corev1"
)

func CheckNodeUnschedulable(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	fmt.Println("[step 1-] Filtering > CheckNodeUnschedulable")

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
	if *resource.AvailableNodeCount == 0 {
		message := fmt.Sprintf("pod (%s) failed to fit in any node", newPod.Pod.ObjectMeta.Name)
		log.Println(message)
		event := postevent.MakeNoNodeEvent(newPod, message)
		err := postevent.PostEvent(event)
		if err != nil {
			fmt.Println("CheckNodeUnschedulable error: ", err)
			return err
		}
		return err
	}

	return nil
}
