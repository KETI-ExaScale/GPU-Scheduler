package predicates

import (
	"fmt"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
)

func PodFitsResourcesAndGPU(nodeInfoList []*resource.NodeInfo, newPod *corev1.Pod) error {
	fmt.Println(" 1-1. PodFitsResourcesAndGPU")

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if nodeinfo.NodeName == "temp" {
				nodeinfo.FilterNode()
			}
		}
	}

	//no node to allocate
	if *resource.AvailableNodeCount == 0 {
		message := fmt.Sprintf("pod (%s) failed to fit in any node", newPod.ObjectMeta.Name)
		event := postevent.MakeNoNodeEvent(newPod, message)
		err := postevent.PostEvent(event)
		if err != nil {
			fmt.Println("podFitsResourcesAndGPU error: ", err)
			return err
		}
	}

	return nil
}
