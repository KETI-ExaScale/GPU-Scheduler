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

package predicates

import (
	"fmt"

	resource "gpu-scheduler/resourceinfo"
)

func Filtering(newPod *resource.Pod, nodeInfoList []*resource.NodeInfo) ([]*resource.NodeInfo, error) {
	fmt.Println("[step 1] Filtering statge")

	//debugging
	fmt.Print(" |Before Filtering Nodes| ")
	for i, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if i == 0 {
				fmt.Print(nodeinfo.NodeName)
				continue
			}
			fmt.Print(" , ", nodeinfo.NodeName)
		}
	}
	fmt.Println()

	//1. PodFitsHost
	err := PodFitsHost(nodeInfoList, newPod)
	if err != nil {
		fmt.Println("Filtering>PodFitsHost error: ", err)
		return nil, err
	}

	//2. CheckGPUAvailable
	err = CheckGPUAvailable(nodeInfoList, newPod)
	if err != nil {
		fmt.Println("Filtering>CheckGPUAvailable error: ", err)
		return nil, err
	}

	//3. PodFitsResourcesAndGPU
	err = PodFitsRequestedResources(nodeInfoList, newPod)
	if err != nil {
		fmt.Println("Filtering>PodFitsResourcesAndGPU error: ", err)
		return nil, err
	}

	//4. PodFitsHostPorts
	err = PodFitsHostPorts(nodeInfoList, newPod)
	if err != nil {
		fmt.Println("Filtering>PodFitsHostPorts error: ", err)
		return nil, err
	}

	//5. MatchNodeSelector
	err = MatchNodeSelector(nodeInfoList, newPod)
	if err != nil {
		fmt.Println("Filtering>MatchNodeSelector error: ", err)
		return nil, err
	}

	//debugging
	fmt.Print(" |After Filtering Nodes| ")
	for i, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if i == 0 {
				fmt.Print(nodeinfo.NodeName)
				continue
			}
			fmt.Print(" , ", nodeinfo.NodeName)
		}
	}
	fmt.Println()

	return nodeInfoList, nil

}
