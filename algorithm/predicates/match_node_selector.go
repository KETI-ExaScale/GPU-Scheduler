package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func MatchNodeSelector() error {
	if config.Filtering {
		fmt.Println("[step 1-4] Filtering > MatchNodeSelector")
	}

	//NodeSelector O
	if len(resource.NewPod.Pod.Spec.NodeSelector) > 0 {
		for _, nodeinfo := range resource.NodeInfoList {
			if !nodeinfo.IsFiltered {
				for key, pod_value := range resource.NewPod.Pod.Spec.NodeSelector {
					if node_value, ok := nodeinfo.Node.Labels[key]; ok {
						if pod_value == node_value {
							continue
						}
					}
					nodeinfo.FilterNode()
					break
				}
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode() {
		return errors.New("<Failed Stage> match_node_selector")
	}
	return nil
}
