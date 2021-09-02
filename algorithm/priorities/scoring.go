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

	corev1 "k8s.io/api/core/v1"
)

type NodePrice struct {
	BestNode  *resource.NodeInfo
	NodeScore float64
}

func Scoring(nodeInfoList []*resource.NodeInfo, newPod *corev1.Pod) (*resource.NodeInfo, error) {
	fmt.Println("2. Scoring Stage")

	var bestPriceNode *NodePrice

	//debugging
	fmt.Print("-Before Scoring Nodes")
	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			fmt.Print(" | ", nodeinfo.NodeName, "=", nodeinfo.NodeScore)
		}
	}
	fmt.Println()

	err := MetricBasedScoring(nodeInfoList)
	if err != nil {
		fmt.Println("scoring>metricBasedScoring error: ", err)
		return &resource.NodeInfo{}, err
	}

	//debugging
	fmt.Print("-After Scoring Nodes")
	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			fmt.Print(" | ", nodeinfo.NodeName, "=", nodeinfo.NodeScore)
		}
	}
	fmt.Println()

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if bestPriceNode == nil {
				bestPriceNode = &NodePrice{nodeinfo, nodeinfo.NodeScore}
				continue
			}
			if nodeinfo.NodeScore > bestPriceNode.NodeScore {
				bestPriceNode.BestNode = nodeinfo
				bestPriceNode.NodeScore = nodeinfo.NodeScore
			}
		}
	}

	if bestPriceNode == nil {
		bestPriceNode = &NodePrice{nodeInfoList[0], 0}
	}

	fmt.Println("BestNode: ", bestPriceNode.BestNode.NodeName)

	return bestPriceNode.BestNode, nil
}
