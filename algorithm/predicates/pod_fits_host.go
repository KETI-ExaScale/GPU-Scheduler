package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func PodFitsHost(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-1] Filtering > PodFitsHost")
	}

	//NodeName O
	if len(newPod.Pod.Spec.NodeName) != 0 {
		for _, nodeinfo := range nodeInfoList {
			if !nodeinfo.IsFiltered {
				if newPod.Pod.Spec.NodeName != nodeinfo.NodeName {
					nodeinfo.FilterNode()
				}
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> pod_fits_host")
	}

	return nil
}
