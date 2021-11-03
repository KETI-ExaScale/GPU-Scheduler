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
	"math"

	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
)

func TaintToleration(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Debugg {
		fmt.Println("[step 2-5] Scoring > TaintToleration")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			var tolerationsPreferNoSchedule []v1.Toleration
			tolerationsPreferNoSchedule = getAllTolerationPreferNoSchedule(newPod.Pod.Spec.Tolerations)

			taintsCount := float64(countIntolerableTaintsPreferNoSchedule(nodeinfo.Node.Spec.Taints, tolerationsPreferNoSchedule))
			nodeScore := 100 - taintsCount/taintsCount*100 //하나라도 있으면 0점...?
			nodeinfo.NodeScore = int(math.Round(nodeScore * float64(1/config.N)))
		}
	}

	return nil
}

func getAllTolerationPreferNoSchedule(tolerations []corev1.Toleration) (tolerationList []corev1.Toleration) {
	for _, toleration := range tolerations {
		// Empty effect means all effects which includes PreferNoSchedule, so we need to collect it as well.
		if len(toleration.Effect) == 0 || toleration.Effect == corev1.TaintEffectPreferNoSchedule {
			tolerationList = append(tolerationList, toleration)
		}
	}
	return
}

func countIntolerableTaintsPreferNoSchedule(taints []corev1.Taint, tolerations []corev1.Toleration) (intolerableTaints int) {
	for _, taint := range taints {
		// check only on taints that have effect PreferNoSchedule
		if taint.Effect != corev1.TaintEffectPreferNoSchedule {
			continue
		}

		if !TolerationsTolerateTaint(tolerations, &taint) {
			intolerableTaints++
		}
	}
	return
}

func TolerationsTolerateTaint(tolerations []v1.Toleration, taint *v1.Taint) bool {
	for i := range tolerations {
		if tolerations[i].ToleratesTaint(taint) {
			return true
		}
	}
	return false
}
