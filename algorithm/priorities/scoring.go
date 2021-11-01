// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package priorities

import (
	"fmt"

	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func Scoring(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) ([]*resource.NodeInfo, error) {
	if config.Debugg {
		fmt.Println("[step 2] Scoring Stage")

		//debugging
		fmt.Print(" |Before Scoring Nodes| ")
		for i, nodeinfo := range nodeInfoList {
			if !nodeinfo.IsFiltered {
				if i == 0 {
					fmt.Print(nodeinfo.NodeName, "=", nodeinfo.NodeScore)
					continue
				}
				fmt.Print(" , ", nodeinfo.NodeName, "=", nodeinfo.NodeScore)
			}
		}
		fmt.Println()
	}

	//1. LeastRequestedResource
	err := LeastRequestedResource(nodeInfoList, newPod)
	if err != nil {
		fmt.Println("scoring>LeastRequestedResource error: ", err)
		return nodeInfoList, err
	}

	// //debugging
	// for _, node := range nodeInfoList {
	// 	for _, gpu := range node.GPUMetrics {
	// 		fmt.Println("[gpuscore]", gpu.UUID, gpu.GPUScore)
	// 	}
	// }

	//2. CheckGPUAvailable
	err = MetricBasedScoring(nodeInfoList, newPod)
	if err != nil {
		fmt.Println("scoring>metricBasedScoring error: ", err)
		return nodeInfoList, err
	}

	//debugging
	fmt.Print(" |After Scoring Nodes| ")
	for i, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if i == 0 {
				fmt.Print(nodeinfo.NodeName, "=", nodeinfo.NodeScore)
				continue
			}
			fmt.Print(" , ", nodeinfo.NodeName, "=", nodeinfo.NodeScore)
		}
	}
	fmt.Println()

	return nodeInfoList, nil
}
