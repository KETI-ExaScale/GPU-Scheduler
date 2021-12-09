package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func CheckNodeReserved(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-12] Filtering > CheckNodeReserved")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if nodeinfo.Node.Annotations["reserved"] != "" {
				nodeinfo.FilterNode()
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> check_node_reserved")
	}

	return nil
}
