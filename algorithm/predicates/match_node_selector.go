package predicates

import (
	"fmt"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"
	"log"
)

func MatchNodeSelector(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	fmt.Println("[step 1-5] Filtering > MatchNodeSelector")

	//NodeSelector O
	if len(newPod.Pod.Spec.NodeSelector) != 0 {
		for _, nodeinfo := range nodeInfoList {
			if !nodeinfo.IsFiltered {
				for key, pod_value := range newPod.Pod.Spec.NodeSelector {
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
	if *resource.AvailableNodeCount == 0 {
		message := fmt.Sprintf("pod (%s) failed to fit in any node", newPod.Pod.ObjectMeta.Name)
		log.Println(message)
		event := postevent.MakeNoNodeEvent(newPod, message)
		err := postevent.PostEvent(event)
		if err != nil {
			fmt.Println("PodFitsResourcesAndGPU error: ", err)
			return err
		}
		return err
	}

	return nil
}
