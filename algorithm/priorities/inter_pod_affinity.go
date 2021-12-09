// // Copyright 2016 Google Inc. All Rights Reserved.
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //     http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.

package priorities

// import (
// 	"fmt"
// 	"math"

// 	"gpu-scheduler/config"
// 	resource "gpu-scheduler/resourceinfo"
// )

// type topologyPairToScore map[string]map[string]int64

// func InterPodAffinity(newPod *resource.Pod) error {
// 	if config.Scoring {
// 		fmt.Println("[step 2-7] Scoring > InterPodAffinity")
// 	}

// 	for _, nodeinfo := range resource.NodeInfoList {
// 		if !nodeinfo.IsFiltered {
// 			nodeScore, weight := float64(0), int64(0)

// 			var topologyScore topologyPairToScore
// 			if priorityMeta, ok := meta.(*priorityMetadata); ok {
// 				topologyScore = priorityMeta.topologyScore
// 			}

// 			for tpKey, tpValues := range topologyScore {
// 				if v, exist := nodeinfo.Node.Labels[tpKey]; exist {
// 					weight += tpValues[v]
// 				}
// 			}

// 			nodeScore = float64(weight)
// 			nodeinfo.NodeScore += int(math.Round(nodeScore * float64(1/config.N)))
// 		}
// 	}

// 	return nil
// }
