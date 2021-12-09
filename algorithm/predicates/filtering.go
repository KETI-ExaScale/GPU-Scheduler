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

func Filtering() error {
	if config.Debugg {
		fmt.Println("[step 1] Filtering statge")

		fmt.Println("<Before Filtering>")
		fmt.Println("         NodeName         |                GPU")

		for _, nodeinfo := range resource.NodeInfoList {
			temp := true
			if !nodeinfo.IsFiltered {
				fmt.Printf(" %-25v", nodeinfo.NodeName)

				for _, gpu := range nodeinfo.GPUMetrics {
					if !gpu.IsFiltered {
						if temp {
							fmt.Printf("|{%s} \n", gpu.UUID)
							temp = false
						} else {
							fmt.Printf("\t\t\t  |{%s} \n", gpu.UUID)
						}
					}
				}
			}
		}
	}

	//1. PodFitsHost
	err := PodFitsHost()
	if err != nil {
		fmt.Println(err)
		return err
	}

	//2. CheckNodeUnschedulable
	err = CheckNodeUnschedulable()
	if err != nil {
		fmt.Println(err)
		return err
	}

	//3. PodFitsHostPorts
	err = PodFitsHostPorts()
	if err != nil {
		fmt.Println(err)
		return err
	}

	//4. MatchNodeSelector
	err = MatchNodeSelector()
	if err != nil {
		fmt.Println(err)
		return err
	}

	//5. PodToleratesNodeTaints
	err = PodToleratesNodeTaints()
	if err != nil {
		fmt.Println(err)
		return err
	}

	//6. CheckGPUAvailable
	err = CheckGPUAvailable()
	if err != nil {
		fmt.Println(err)
		return err
	}

	//7. PodFitsResourcesAndGPU
	err = PodFitsResources()
	if err != nil {
		fmt.Println(err)
		return err
	}

	//8. NoDiskConflict
	err = NoDiskConflict()
	if err != nil {
		fmt.Println(err)
		return err
	}

	// //9. MaxCSIVolumeCount
	// err = MaxCSIVolumeCount()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return err
	// }

	// //10. NoVolumeZoneConflict
	// err = NoVolumeZoneConflict()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return err
	// }

	// //11. CheckVolumeBinding
	// err = CheckVolumeBinding()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return err
	// }

	// //12. CheckNodeReserved
	// err = CheckNodeReserved()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return err
	// }

	if config.Debugg {
		fmt.Println("<After Filtering>")
		fmt.Println("         NodeName         |                GPU")

		for _, nodeinfo := range resource.NodeInfoList {
			temp := true
			if !nodeinfo.IsFiltered {
				fmt.Printf(" %-25v", nodeinfo.NodeName)

				for _, gpu := range nodeinfo.GPUMetrics {
					if !gpu.IsFiltered {
						if temp {
							fmt.Printf("|{%s} \n", gpu.UUID)
							temp = false
						} else {
							fmt.Printf("\t\t\t  |{%s} \n", gpu.UUID)
						}

					}
				}
			}
		}

	}

	return nil

}
