package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func PodFitsHost() error {
	if config.Filtering {
		fmt.Println("[step 1-1] Filtering > PodFitsHost")
	}

	//NodeName O
	if len(resource.NewPod.Pod.Spec.NodeName) > 0 {
		for _, nodeinfo := range resource.NodeInfoList {
			if !nodeinfo.IsFiltered {
				if resource.NewPod.Pod.Spec.NodeName != nodeinfo.NodeName {
					nodeinfo.FilterNode()
				}
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode() {
		return errors.New("<Failed Stage> pod_fits_host")
	}

	return nil
}
