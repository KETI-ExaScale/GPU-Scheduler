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

	resource "gpu-scheduler/resourceinfo"
)

// type NodePrice struct {
// 	BestNode  *resource.NodeInfo
// 	NodeScore int
// }

func Scoring(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) ([]*resource.NodeInfo, error) {
	fmt.Println("[step 2] Scoring Stage")

	//var bestPriceNode *NodePrice

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

	//1. LeastGPUMemory
	err := LeastGPUMemory(nodeInfoList, newPod)
	if err != nil {
		fmt.Println("scoring>metricBasedScoring error: ", err)
		return nodeInfoList, err
	}

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
