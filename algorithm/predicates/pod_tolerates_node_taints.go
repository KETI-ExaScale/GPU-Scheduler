package predicates

import (
	"fmt"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"
	"log"
)

func PodToleratesNodeTaints(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	fmt.Println("[step 1-6] Filtering > PodToleratesNodeTaints")

	//NodeSelector O
	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			for _, taint := range nodeinfo.Node.Spec.Taints {
				if taint.Effect != "PreferNoSchedule" {
					canTolerate := false
					for _, toleration := range newPod.Pod.Spec.Tolerations {
						if taint.Key == toleration.Key {
							// if toleration.Operator == "Equal"{
							// 	if taint.Effect == toleration.Effect && taint.Value == toleration.Value{
							// 		canTolerate = true
							// 	}
							// }else{
							// 	if taint.Effect == toleration.Effect{
							//		canTolerate = true
							//}
							// }
							if ((toleration.Operator == "Equal") && (taint.Effect == toleration.Effect) && (taint.Value == toleration.Value)) ||
								((toleration.Operator == "Exists") && (taint.Effect == toleration.Effect)) {
								canTolerate = true
							}
						}
						if canTolerate {
							break
						}
					}
					if !canTolerate {
						nodeinfo.FilterNode()
						break
					}
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
