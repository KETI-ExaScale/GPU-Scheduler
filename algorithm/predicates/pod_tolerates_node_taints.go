package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func PodToleratesNodeTaints(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-5] Filtering > PodToleratesNodeTaints")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			for _, taint := range nodeinfo.Node.Spec.Taints {
				tolerated := false
				for _, toleration := range newPod.Pod.Spec.Tolerations {
					if toleration.ToleratesTaint(&taint) {
						tolerated = true
						break
					}
				}
				if !tolerated {
					nodeinfo.FilterNode()
					break
				}
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> pod_tolerates_node_taints")
	}

	return nil
}
