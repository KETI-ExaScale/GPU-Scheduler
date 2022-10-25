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

	v1 "k8s.io/api/core/v1"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
)

type NodeAffinity struct {
	addedPrefSchedTerms *nodeaffinity.PreferredSchedulingTerms
}

func (pl NodeAffinity) Name() string {
	return "NodeAffinity"
}

func (pl NodeAffinity) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("S#1. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			r.KETI_LOG_L1(fmt.Sprintf("-node {%s} score: %d", nodeName, nodeInfo.PluginResult.NodeScore))
		}
	}
}

func (pl NodeAffinity) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {

	//PreScore
	preferredNodeAffinity, err := getPodPreferredNodeAffinity(newPod.Pod)
	if err != nil {
		return
	}
	state := &preScoreState1{
		preferredNodeAffinity: preferredNodeAffinity,
	}

	//Score
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {

			var count int64
			if pl.addedPrefSchedTerms != nil {
				count += pl.addedPrefSchedTerms.Score(nodeinfo.Node())
			}

			if state.preferredNodeAffinity != nil {
				count += state.preferredNodeAffinity.Score(nodeinfo.Node())
			}
			nodeinfo.PluginResult.NodeScore += int(math.Round(float64(count)))
		}
	}

}

func getPodPreferredNodeAffinity(pod *v1.Pod) (*nodeaffinity.PreferredSchedulingTerms, error) {
	affinity := pod.Spec.Affinity
	if affinity != nil && affinity.NodeAffinity != nil && affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
		return nodeaffinity.NewPreferredSchedulingTerms(affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	}
	return nil, nil
}

// func (pl NodeAffinity) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {

// 	affinity := newPod.Pod.Spec.Affinity

// 	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
// 		if !nodeinfo.PluginResult.IsFiltered {

// 			nodeScore, count, weight := int(0), 0, int32(0)
// 			if affinity != nil && affinity.NodeAffinity != nil && affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
// 				for i := range affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {

// 					preferredSchedulingTerm := &affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i]
// 					if preferredSchedulingTerm.Weight == 0 {
// 						continue
// 					}

// 					nodeSelector, err := NodeSelectorRequirementsAsSelector(preferredSchedulingTerm.Preference.MatchExpressions)
// 					if err != nil {
// 						return // 여기 원래 return err인거 처리
// 					}

// 					if nodeSelector.Matches(labels.Set(nodeinfo.Node().Labels)) {
// 						count += 1
// 						weight += preferredSchedulingTerm.Weight
// 					}
// 				}
// 			}

// 			if count > 0 {
// 				nodeScore = int(weight) * 1 / int(count)
// 			}

// 			nodeinfo.PluginResult.NodeScore += int(math.Round(float64(nodeScore / r.Ns)))
// 		}
// 	}

// }

// func NodeSelectorRequirementsAsSelector(nsm []corev1.NodeSelectorRequirement) (labels.Selector, error) {
// 	if len(nsm) == 0 {
// 		return labels.Nothing(), nil
// 	}
// 	selector := labels.NewSelector()
// 	for _, expr := range nsm {
// 		var op selection.Operator
// 		switch expr.Operator {
// 		case corev1.NodeSelectorOpIn:
// 			op = selection.In
// 		case corev1.NodeSelectorOpNotIn:
// 			op = selection.NotIn
// 		case corev1.NodeSelectorOpExists:
// 			op = selection.Exists
// 		case corev1.NodeSelectorOpDoesNotExist:
// 			op = selection.DoesNotExist
// 		case corev1.NodeSelectorOpGt:
// 			op = selection.GreaterThan
// 		case corev1.NodeSelectorOpLt:
// 			op = selection.LessThan
// 		default:
// 			return nil, fmt.Errorf("%q is not a valid node selector operator", expr.Operator)
// 		}
// 		r, err := labels.NewRequirement(expr.Key, op, expr.Values)
// 		if err != nil {
// 			return nil, err
// 		}
// 		selector = selector.Add(*r)
// 	}
// 	return selector, nil
// }
