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

	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func Filtering(newPod *resource.Pod, nodeInfoList []*resource.NodeInfo) ([]*resource.NodeInfo, error) {
	if config.Debugg {
		fmt.Println("[step 1] Filtering statge")

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
	}

	//1. PodFitsHost
	err := PodFitsHost(nodeInfoList, newPod)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//2. CheckNodeUnschedulable
	err = CheckNodeUnschedulable(nodeInfoList, newPod)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//3. PodFitsHostPorts
	err = PodFitsHostPorts(nodeInfoList, newPod)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//4. MatchNodeSelector
	err = MatchNodeSelector(nodeInfoList, newPod)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//5. PodToleratesNodeTaints
	err = PodToleratesNodeTaints(nodeInfoList, newPod)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//6. CheckGPUAvailable
	err = CheckGPUAvailable(nodeInfoList, newPod)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//7. PodFitsResourcesAndGPU
	err = PodFitsResources(nodeInfoList, newPod)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	//8. NoDiskConflict
	err = NoDiskConflict(nodeInfoList, newPod)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// //9. MaxCSIVolumeCount
	// err = MaxCSIVolumeCount(nodeInfoList, newPod)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return nil, err
	// }

	// //10. NoVolumeZoneConflict
	// err = NoVolumeZoneConflict(nodeInfoList, newPod)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return nil, err
	// }

	// //11. CheckVolumeBinding
	// err = CheckVolumeBinding(nodeInfoList, newPod)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return nil, err
	// }

	// //12. CheckNodeReserved
	// err = CheckNodeReserved(nodeInfoList, newPod)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return nil, err
	// }

	if config.Debugg {
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
	}

	return nodeInfoList, nil

}
