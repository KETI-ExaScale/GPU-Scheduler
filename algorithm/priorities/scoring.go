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

		fmt.Println("<Before Scoring>")
		fmt.Println("         NodeName         |                GPU")

		for _, nodeinfo := range nodeInfoList {
			temp := true
			if !nodeinfo.IsFiltered {
				fmt.Printf(" {%-17v : %-3v}", nodeinfo.NodeName, nodeinfo.NodeScore)

				for _, gpu := range nodeinfo.GPUMetrics {
					if !gpu.IsFiltered {
						if temp {
							fmt.Printf("|{%v : %v}\n", gpu.UUID, gpu.GPUScore)
							temp = false
						} else {
							fmt.Printf("\t\t\t  |{%v : %v}\n", gpu.UUID, gpu.GPUScore)
						}
					}
				}
			}
		}
	}

	//1. LeastRequestedResource
	err := LeastRequestedResource(nodeInfoList, newPod)
	if err != nil {
		return nodeInfoList, err
	}

	//2. BalancedResourceAllocation
	err = BalancedResourceAllocation(nodeInfoList, newPod)
	if err != nil {
		return nodeInfoList, err
	}

	// //3. ImageLocality
	// err = ImageLocality(nodeInfoList, newPod)
	// if err != nil {
	// 	return nodeInfoList, err
	// }

	//4. NodeAffinity
	err = NodeAffinity(nodeInfoList, newPod)
	if err != nil {
		return nodeInfoList, err
	}

	//5. TaintToleration
	err = TaintToleration(nodeInfoList, newPod)
	if err != nil {
		return nodeInfoList, err
	}

	// //6. SelectorSpread
	// err = SelectorSpread(nodeInfoList, newPod)
	// if err != nil {
	// 	return nodeInfoList, err
	// }

	// //7. InterPodAffinity
	// err = InterPodAffinity(nodeInfoList, newPod)
	// if err != nil {
	// 	return nodeInfoList, err
	// }

	// //8. EvenPodsSpread
	// err = EvenPodsSpread(nodeInfoList, newPod)
	// if err != nil {
	// 	return nodeInfoList, err
	// }

	//9. LeastGPUMemoryUsage
	err = LeastGPUMemoryUsage(nodeInfoList, newPod)
	if err != nil {
		return nodeInfoList, err
	}

	//10. LeastGPUMemoryUtilization
	err = LeastGPUMemoryUtilization(nodeInfoList, newPod)
	if err != nil {
		return nodeInfoList, err
	}

	//11. LeastAllocatedPodGPU
	err = LeastAllocatedPodGPU(nodeInfoList, newPod)
	if err != nil {
		return nodeInfoList, err
	}

	if config.Debugg {
		fmt.Println("<After Scoring>")
		fmt.Println("         NodeName         |                GPU")

		for _, nodeinfo := range nodeInfoList {
			temp := true
			if !nodeinfo.IsFiltered {
				fmt.Printf(" {%-17v : %-3v}", nodeinfo.NodeName, nodeinfo.NodeScore)

				for _, gpu := range nodeinfo.GPUMetrics {
					if !gpu.IsFiltered {
						if temp {
							fmt.Printf("|{%v : %v}\n", gpu.UUID, gpu.GPUScore)
							temp = false
						} else {
							fmt.Printf("\t\t\t  |{%v : %v}\n", gpu.UUID, gpu.GPUScore)
						}
					}
				}
			}
		}
	}

	return nodeInfoList, nil
}
