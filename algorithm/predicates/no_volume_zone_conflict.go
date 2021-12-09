package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func NoVolumeZoneConflict() error {
	if config.Filtering {
		fmt.Println("[step 1-10] Filtering > NoVolumeZoneConflict")
	}

	for _, nodeinfo := range resource.NodeInfoList {
		if !nodeinfo.IsFiltered {

		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode() {
		return errors.New("<Failed Stage> no_volume_zone_conflict")
	}

	return nil
}
