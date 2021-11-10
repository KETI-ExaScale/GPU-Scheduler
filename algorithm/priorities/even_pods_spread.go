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
// 	"go/build/constraint"
// 	"math"

// 	"gpu-scheduler/config"
// 	resource "gpu-scheduler/resourceinfo"
// )

// func EvenPodsSpread(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
// 	if config.Scoring {
// 		fmt.Println("[step 2-8] Scoring > EvenPodsSpread")
// 	}

// 	for _, nodeinfo := range nodeInfoList {
// 		if !nodeinfo.IsFiltered {
// 			nodeScore := float64(0)

// 			nodeinfo.NodeScore += int(math.Round(nodeScore * float64(1/config.N)))
// 		}
// 	}

// 	return nil
// }

// func filterSoftTopologySpreadConstraints(constraints []corev1.TopologySpreadConstraint) ([]topologySpreadConstraint, error){
// 	var r []topologySpreadConstraint
// 	for _, c := range constraints{
// 		if c.WhenUnsta
// 	}
// }
