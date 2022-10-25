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
	"sync/atomic"

	r "gpu-scheduler/resourceinfo"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersv1 "k8s.io/client-go/listers/core/v1"
)

type InterPodAffinity struct {
	nsLister listersv1.NamespaceLister
	args     InterPodAffinityArgs
}

type InterPodAffinityArgs struct {
	metav1.TypeMeta
	HardPodAffinityWeight int32
}

func (pl InterPodAffinity) Name() string {
	return "InterPodAffinity"
}

func (pl InterPodAffinity) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("S#4. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			r.KETI_LOG_L1(fmt.Sprintf("-node {%s} score: %d\n", nodeName, nodeInfo.PluginResult.NodeScore))
		}
	}
}

func (pl *InterPodAffinity) PreScore(cache *r.NodeCache, newPod *r.QueuedPodInfo) (*preScoreState4, error) {
	affinity := newPod.Pod.Spec.Affinity
	hasPreferredAffinityConstraints := affinity != nil && affinity.PodAffinity != nil && len(affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0
	hasPreferredAntiAffinityConstraints := affinity != nil && affinity.PodAntiAffinity != nil && len(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0

	state := &preScoreState4{
		topologyScore: make(map[string]map[string]int64),
	}

	state.podInfo = newPod.PodInfo

	for i := range state.podInfo.PreferredAffinityTerms {
		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&state.podInfo.PreferredAffinityTerms[i].AffinityTerm); err != nil {
			return nil, fmt.Errorf("updating PreferredAffinityTerms: %w", err)
		}
	}
	for i := range state.podInfo.PreferredAntiAffinityTerms {
		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&state.podInfo.PreferredAntiAffinityTerms[i].AffinityTerm); err != nil {
			return nil, fmt.Errorf("updating PreferredAntiAffinityTerms: %w", err)
		}
	}
	state.namespaceLabels = GetNamespaceLabelsSnapshot(newPod.Pod.Namespace, pl.nsLister)

	topoScores := make([]scoreMap, cache.AvailableNodeCount)
	index := int32(-1)
	for _, nodeinfo := range cache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			podsToProcess := nodeinfo.PodsWithAffinity
			if hasPreferredAffinityConstraints || hasPreferredAntiAffinityConstraints {
				podsToProcess = nodeinfo.Pods
			}

			topoScore := make(scoreMap)
			for _, existingPod := range podsToProcess {
				pl.processExistingPod(state, existingPod, nodeinfo, newPod.Pod, topoScore)
			}
			if len(topoScore) > 0 {
				topoScores[atomic.AddInt32(&index, 1)] = topoScore
			}
		}
	}

	for i := 0; i <= int(index); i++ {
		state.topologyScore.append(topoScores[i])
	}

	return state, nil
}

func (pl InterPodAffinity) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	// state, err := pl.PreScore(nodeInfoCache, newPod)
	// if err != nil {
	// 	fmt.Println("<error> InterPodAffinity PreScore Error - ", err)
	// 	return
	// }

	// for _, nodeinfo := range nodeInfoCache.NodeInfoList {
	// 	if !nodeinfo.PluginResult.IsFiltered {
	// 		var score int64
	// 		for tpKey, tpValues := range state.topologyScore {
	// 			if v, exist := nodeinfo.Node().Labels[tpKey]; exist {
	// 				score += tpValues[v]
	// 			}
	// 		}
	//		fmt.Println("scoring>inter_pod_affinity: ", score, "scode, node: ", nodeinfo.Node().Name)
	// 		nodeinfo.PluginResult.NodeScore += int(float64(score))
	// 	}
	// }

}

func (pl *InterPodAffinity) mergeAffinityTermNamespacesIfNotEmpty(at *r.AffinityTerm) error {
	if at.NamespaceSelector.Empty() {
		return nil
	}
	ns, err := pl.nsLister.List(at.NamespaceSelector)
	if err != nil {
		return err
	}
	for _, n := range ns {
		at.Namespaces.Insert(n.Name)
	}
	at.NamespaceSelector = labels.Nothing()
	return nil
}

func GetNamespaceLabelsSnapshot(ns string, nsLister listersv1.NamespaceLister) (nsLabels labels.Set) {
	podNS, err := nsLister.Get(ns)
	if err == nil {
		// Create and return snapshot of the labels.
		return labels.Merge(podNS.Labels, nil)
	}
	return
}

func (pl *InterPodAffinity) processExistingPod(
	state *preScoreState4,
	existingPod *r.PodInfo,
	existingPodNodeInfo *r.NodeInfo,
	incomingPod *v1.Pod,
	topoScore scoreMap,
) {
	existingPodNode := existingPodNodeInfo.Node()
	if len(existingPodNode.Labels) == 0 {
		return
	}

	topoScore.processTerms(state.podInfo.PreferredAffinityTerms, existingPod.Pod, nil, existingPodNode, 1)

	topoScore.processTerms(state.podInfo.PreferredAntiAffinityTerms, existingPod.Pod, nil, existingPodNode, -1)

	if pl.args.HardPodAffinityWeight > 0 && len(existingPodNode.Labels) != 0 {
		for _, t := range existingPod.RequiredAffinityTerms {
			topoScore.processTerm(&t, pl.args.HardPodAffinityWeight, incomingPod, state.namespaceLabels, existingPodNode, 1)
		}
	}

	topoScore.processTerms(existingPod.PreferredAffinityTerms, incomingPod, state.namespaceLabels, existingPodNode, 1)

	topoScore.processTerms(existingPod.PreferredAntiAffinityTerms, incomingPod, state.namespaceLabels, existingPodNode, -1)
}

func (m scoreMap) append(other scoreMap) {
	for topology, oScores := range other {
		scores := m[topology]
		if scores == nil {
			m[topology] = oScores
			continue
		}
		for k, v := range oScores {
			scores[k] += v
		}
	}
}

func (m scoreMap) processTerms(terms []r.WeightedAffinityTerm, pod *v1.Pod, nsLabels labels.Set, node *v1.Node, multiplier int32) {
	for _, term := range terms {
		m.processTerm(&term.AffinityTerm, term.Weight, pod, nsLabels, node, multiplier)
	}
}

func (m scoreMap) processTerm(term *r.AffinityTerm, weight int32, pod *v1.Pod, nsLabels labels.Set, node *v1.Node, multiplier int32) {
	if term.Matches(pod, nsLabels) {
		if tpValue, tpValueExist := node.Labels[term.TopologyKey]; tpValueExist {
			if m[term.TopologyKey] == nil {
				m[term.TopologyKey] = make(map[string]int64)
			}
			m[term.TopologyKey][tpValue] += int64(weight * multiplier)
		}
	}
}
