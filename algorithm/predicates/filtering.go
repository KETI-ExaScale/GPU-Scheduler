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

	corev1 "k8s.io/api/core/v1"
)

func Filtering(newPod *corev1.Pod) ([]*resource.NodeInfo, error) {
	fmt.Println("1. Filtering statge")

	//새 파드 필터링 전 노드 정보 업데이트
	var NodeInfoList []*resource.NodeInfo
	var NodeMetricList []*resource.NodeMetric
	NodeInfoList, NodeMetricList, err := resource.NodeUpdate(NodeInfoList, NodeMetricList)
	if err != nil {
		fmt.Println("Filtering>nodeUpdate error: ", err)
		return nil, err
	}

	//debugging
	fmt.Print("-Before Filtering Nodes")
	for _, nodeinfo := range NodeInfoList {
		if !nodeinfo.IsFiltered {
			fmt.Print(" | ", nodeinfo.NodeName)
		}
	}
	fmt.Println()

	//1. PodFitsResourcesAndGPU
	err = PodFitsResourcesAndGPU(NodeInfoList, newPod)
	if err != nil {
		fmt.Println("Filtering>PodFitsResourcesAndGPU error: ", err)
		return nil, err
	}

	//debugging
	fmt.Print("-After Filtering Nodes")
	for _, nodeinfo := range NodeInfoList {
		if !nodeinfo.IsFiltered {
			fmt.Print(" | ", nodeinfo.NodeName)
		}
	}
	fmt.Println()

	return NodeInfoList, nil

}
