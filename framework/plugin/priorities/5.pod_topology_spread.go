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
	"log"
	"math"
	"sync/atomic"

	r "gpu-scheduler/resourceinfo"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
)

type PodTopologySpread struct {
	systemDefaulted    bool
	defaultConstraints []v1.TopologySpreadConstraint
	services           corelisters.ServiceLister
	replicationCtrls   corelisters.ReplicationControllerLister
	replicaSets        appslisters.ReplicaSetLister
	statefulSets       appslisters.StatefulSetLister
}

type ptsPreScoreState struct {
	Constraints               []topologySpreadConstraint
	IgnoredNodes              sets.String
	TopologyPairToPodCounts   map[topologyPair]*int64
	TopologyNormalizingWeight []float64
}
type topologySpreadConstraint struct {
	MaxSkew     int32
	TopologyKey string
	Selector    labels.Selector
	MinDomains  int32
}
type topologyPair struct {
	key   string
	value string
}

func PodTopologySpread_() PodTopologySpread {
	hostConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)
	informerFactory := informers.NewSharedInformerFactory(hostKubeClient, 0)

	services := informerFactory.Core().V1().Services().Lister()
	replicationCtrls := informerFactory.Core().V1().ReplicationControllers().Lister()
	replicaSets := informerFactory.Apps().V1().ReplicaSets().Lister()
	statefulSets := informerFactory.Apps().V1().StatefulSets().Lister()

	return PodTopologySpread{
		systemDefaulted:    r,
		defaultConstraints: r,
		services:           services,
		replicationCtrls:   replicationCtrls,
		replicaSets:        replicaSets,
		statefulSets:       statefulSets,
	}
}

func (pl PodTopologySpread) Name() string {
	return "PodTopologySpread"
}

func (pl PodTopologySpread) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("S#5. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			r.KETI_LOG_L1(fmt.Sprintf("-node {%s} score: %d", nodeName, nodeInfo.PluginResult.NodeScore))
		}
	}
}

func (pl *PodTopologySpread) initPreScoreState(s *ptsPreScoreState, pod *v1.Pod, cache *r.NodeCache, requireAllTopologies bool) error {
	var err error

	if len(pod.Spec.TopologySpreadConstraints) > 0 {
		s.Constraints, err = filterTopologySpreadConstraints(pod.Spec.TopologySpreadConstraints, v1.ScheduleAnyway)
		if err != nil {
			return fmt.Errorf("obtaining pod's soft topology spread constraints: %w", err)
		}
	} else {
		s.Constraints, err = pl.buildDefaultConstraints(pod, v1.ScheduleAnyway)
		if err != nil {
			return fmt.Errorf("setting default soft topology spread constraints: %w", err)
		}
	}

	if len(s.Constraints) == 0 {
		return nil
	}
	topoSize := make([]int, len(s.Constraints))
	for _, node := range cache.NodeInfoList {
		if !node.PluginResult.IsFiltered {
			if requireAllTopologies && !nodeLabelsMatchSpreadConstraints(node.Node().Labels, s.Constraints) {
				// Nodes which don't have all required topologyKeys present are ignored
				// when scoring later.
				s.IgnoredNodes.Insert(node.Node().Name)
				continue
			}
			for i, constraint := range s.Constraints {
				// per-node counts are calculated during Score.
				if constraint.TopologyKey == v1.LabelHostname {
					continue
				}
				pair := topologyPair{key: constraint.TopologyKey, value: node.Node().Labels[constraint.TopologyKey]}
				if s.TopologyPairToPodCounts[pair] == nil {
					s.TopologyPairToPodCounts[pair] = new(int64)
					topoSize[i]++
				}
			}
		}
	}

	s.TopologyNormalizingWeight = make([]float64, len(s.Constraints))
	for i, c := range s.Constraints {
		sz := topoSize[i]
		if c.TopologyKey == v1.LabelHostname {
			sz = cache.AvailableNodeCount - len(s.IgnoredNodes)
		}
		s.TopologyNormalizingWeight[i] = topologyNormalizingWeight(sz)
	}
	return nil
}

func (pl *PodTopologySpread) PreScore(cache *r.NodeCache, newPod *r.QueuedPodInfo) (*ptsPreScoreState, error) {
	state := &ptsPreScoreState{
		IgnoredNodes:            sets.NewString(),
		TopologyPairToPodCounts: make(map[topologyPair]*int64),
	}

	requireAllTopologies := len(newPod.Pod.Spec.TopologySpreadConstraints) > 0 || !pl.systemDefaulted
	err := pl.initPreScoreState(state, newPod.Pod, cache, requireAllTopologies)
	if err != nil {
		return nil, fmt.Errorf("calculating preScoreState: %w", err)
	}

	// return if incoming pod doesn't have soft topology spread Constraints.
	if len(state.Constraints) == 0 {
		return state, nil
	}

	// Ignore parsing errors for backwards compatibility.
	requiredNodeAffinity := nodeaffinity.GetRequiredNodeAffinity(newPod.Pod)
	for _, nodeinfo := range cache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			match, _ := requiredNodeAffinity.Match(nodeinfo.Node())
			if !match || (requireAllTopologies && !nodeLabelsMatchSpreadConstraints(nodeinfo.Node().Labels, state.Constraints)) {
				continue
			}

			for _, c := range state.Constraints {
				pair := topologyPair{key: c.TopologyKey, value: nodeinfo.Node().Labels[c.TopologyKey]}
				tpCount := state.TopologyPairToPodCounts[pair]
				if tpCount == nil {
					continue
				}
				count := countPodsMatchSelector(nodeinfo.Pods, c.Selector, newPod.Pod.Namespace)
				atomic.AddInt64(tpCount, int64(count))
			}
		}
	}

	return state, nil
}

func (pl PodTopologySpread) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			state, err := pl.PreScore(nodeInfoCache, newPod)
			if err != nil {
				fmt.Println("<error> InterPodAffinity PreScore Error - ", err)
				return
			}
			if state.IgnoredNodes.Has(nodeinfo.Node().Name) {
				continue //스코어 점수 없음 (최하점)
			}

			var score float64
			for i, c := range state.Constraints {
				if tpVal, ok := nodeinfo.Node().Labels[c.TopologyKey]; ok {
					var cnt int64
					if c.TopologyKey == v1.LabelHostname {
						cnt = int64(countPodsMatchSelector(nodeinfo.Pods, c.Selector, newPod.Pod.Namespace))
					} else {
						pair := topologyPair{key: c.TopologyKey, value: tpVal}
						cnt = *state.TopologyPairToPodCounts[pair]
					}
					score += scoreForCount(cnt, c.MaxSkew, state.TopologyNormalizingWeight[i])
				}
			}
			score = math.Round(score)
			fmt.Println("scoring>pod_topology_spread: ", score, "scode, node: ", nodeinfo.Node().Name)
			nodeinfo.PluginResult.NodeScore += int(score)
		}
	}
}

func scoreForCount(cnt int64, maxSkew int32, tpWeight float64) float64 {
	return float64(cnt)*tpWeight + float64(maxSkew-1)
}

func nodeLabelsMatchSpreadConstraints(nodeLabels map[string]string, constraints []topologySpreadConstraint) bool {
	for _, c := range constraints {
		if _, ok := nodeLabels[c.TopologyKey]; !ok {
			return false
		}
	}
	return true
}

func countPodsMatchSelector(podInfos []*r.PodInfo, selector labels.Selector, ns string) int {
	count := 0
	for _, p := range podInfos {
		// Bypass terminating Pod (see #87621).
		if p.Pod.DeletionTimestamp != nil || p.Pod.Namespace != ns {
			continue
		}
		if selector.Matches(labels.Set(p.Pod.Labels)) {
			count++
		}
	}
	return count
}

func topologyNormalizingWeight(size int) float64 {
	return math.Log(float64(size + 2))
}

func (pl *PodTopologySpread) buildDefaultConstraints(p *v1.Pod, action v1.UnsatisfiableConstraintAction) ([]topologySpreadConstraint, error) {
	constraints, err := filterTopologySpreadConstraints(pl.defaultConstraints, action)
	if err != nil || len(constraints) == 0 {
		return nil, err
	}
	selector := DefaultSelector(p, pl.services, pl.replicationCtrls, pl.replicaSets, pl.statefulSets)
	if selector.Empty() {
		return nil, nil
	}
	for i := range constraints {
		constraints[i].Selector = selector
	}
	return constraints, nil
}

func filterTopologySpreadConstraints(constraints []v1.TopologySpreadConstraint, action v1.UnsatisfiableConstraintAction) ([]topologySpreadConstraint, error) {
	var result []topologySpreadConstraint
	for _, c := range constraints {
		if c.WhenUnsatisfiable == action {
			selector, err := metav1.LabelSelectorAsSelector(c.LabelSelector)
			if err != nil {
				return nil, err
			}
			result = append(result, topologySpreadConstraint{
				MaxSkew:     c.MaxSkew,
				TopologyKey: c.TopologyKey,
				Selector:    selector,
			})
		}
	}
	return result, nil
}
