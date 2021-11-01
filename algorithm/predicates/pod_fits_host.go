package predicates

import (
	"fmt"
	"gpu-scheduler/config"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"
	"log"
)

func PodFitsHost(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Debugg {
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
	if *resource.AvailableNodeCount == 0 {
		message := fmt.Sprintf("pod (%s) failed to fit in any node", newPod.Pod.ObjectMeta.Name)
		log.Println(message)
		event := postevent.MakeNoNodeEvent(newPod, message)
		err := postevent.PostEvent(event)
		if err != nil {
			fmt.Println("PodFitsHost error: ", err)
			return err
		}
		return err
	}

	return nil
}
