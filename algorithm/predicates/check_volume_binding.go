package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func CheckVolumeBinding(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-11] Filtering > CheckVolumeBinding")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {

		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> check_volume_binding")
	}

	return nil
}
