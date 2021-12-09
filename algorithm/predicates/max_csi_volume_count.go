package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func MaxCSIVolumeCount(newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-9] Filtering > MaxCSIVolumeCount")
	}

	for _, nodeinfo := range resource.NodeInfoList {
		if !nodeinfo.IsFiltered {

		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> csi_volume_count")
	}

	return nil
}
