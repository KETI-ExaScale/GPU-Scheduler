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
)

type SelectorSpread struct{}

func (pl SelectorSpread) Name() string {
	return "SelectorSpread"
}

func (pl SelectorSpread) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("#3. ", pl.Name())
	// for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
	// 	if !nodeInfo.PluginResult.IsFiltered {
	// 		fmt.Printf("-node {%s} score: %f\n", nodeName, nodeInfo.PluginResult.NodeScore)
	// 	}
	// }
}

func (pl SelectorSpread) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		nodeScore := 100
		if !nodeinfo.PluginResult.IsFiltered {
			// selector := getSelector(newPod.Pod)
			nodeinfo.PluginResult.NodeScore += math.Round(float64(nodeScore) * float64(1/r.Ns))
		}
	}
}

// func skipSelectorSpread(pod *v1.Pod) bool {
// 	return len(pod.Spec.TopologySpreadConstraints) != 0
// }

// func getSelector(pod *v1.Pod, sl corelisters.ServiceLister, cl corelisters.ReplicationControllerLister, rsl appslisters.ReplicaSetLister, ssl appslisters.StatefulSetLister) labels.Selector {
// 	labelSet := make(labels.Set)
// 	// Since services, RCs, RSs and SSs match the pod, they won't have conflicting
// 	// labels. Merging is safe.

// 	if services, err := sl.GetPodServices(pod); err == nil {
// 		for _, service := range services {
// 			labelSet = labels.Merge(labelSet, service.Spec.Selector)
// 		}
// 	}

// 	if rcs, err := cl.GetPodControllers(pod); err == nil {
// 		for _, rc := range rcs {
// 			labelSet = labels.Merge(labelSet, rc.Spec.Selector)
// 		}
// 	}

// 	selector := labels.NewSelector()
// 	if len(labelSet) != 0 {
// 		selector = labelSet.AsSelector()
// 	}

// 	if rss, err := rsl.GetPodReplicaSets(pod); err == nil {
// 		for _, rs := range rss {
// 			if other, err := metav1.LabelSelectorAsSelector(rs.Spec.Selector); err == nil {
// 				if r, ok := other.Requirements(); ok {
// 					selector = selector.Add(r...)
// 				}
// 			}
// 		}
// 	}

// 	if sss, err := ssl.GetPodStatefulSets(pod); err == nil {
// 		for _, ss := range sss {
// 			if other, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector); err == nil {
// 				if r, ok := other.Requirements(); ok {
// 					selector = selector.Add(r...)
// 				}
// 			}
// 		}
// 	}

// 	return selector
// }
