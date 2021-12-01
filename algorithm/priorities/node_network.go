// 	for _, node := range clusterInfo.Nodes {
// 		if node.UpdateRX == 0 && node.UpdateTX == 0 {
// 			nodeScore = maxScore
// 		} else {
// 			nodeScore = int64((1 / float64(node.UpdateRX+node.UpdateTX)) * float64(maxScore))
// 		}

import (
	"fmt"
	"math"

	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func NodeNetwork(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	// if config.Scoring {
	// 	fmt.Println("[step 2-1] Scoring > LeastRequestedResource")
	// }

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if node.UpdateRX == 0 && node.UpdateTX == 0 {
				nodeScore = maxScore
			} else {
				nodeScore = int64((1 / float64(node.UpdateRX+node.UpdateTX)) * float64(maxScore))
			}
		}
	}

	return nil
}
