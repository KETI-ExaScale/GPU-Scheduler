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

	r "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
)

type TaintToleration struct{}

func (pl TaintToleration) Name() string {
	return "TaintToleration"
}

func (pl TaintToleration) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("#2. ", pl.Name())
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			fmt.Printf("-node {%s} score: %d\n", nodeName, nodeInfo.PluginResult.NodeScore)
		}
	}
}

func (pl TaintToleration) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			nodeScore := 100
			tolerationsPreferNoSchedule := getAllTolerationPreferNoSchedule(newPod.Pod.Spec.Tolerations)
			taintsCount := countIntolerableTaintsPreferNoSchedule(nodeinfo.Node().Spec.Taints, tolerationsPreferNoSchedule)

			if taintsCount > 0 {
				nodeScore = 0
			}

			nodeinfo.PluginResult.NodeScore += int(math.Round(float64(nodeScore / r.Ns)))
		}
	}
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

func TolerationsTolerateTaint(tolerations []corev1.Toleration, taint *corev1.Taint) bool {
	for i := range tolerations {
		if tolerations[i].ToleratesTaint(taint) {
			return true
		}
	}
	return false
}
