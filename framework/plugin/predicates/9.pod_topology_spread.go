package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
	"math"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
	"k8s.io/klog/v2"
)

type PodTopologySpread struct {
	defaultConstraints                  []v1.TopologySpreadConstraint
	services                            corelisters.ServiceLister
	replicationCtrls                    corelisters.ReplicationControllerLister
	replicaSets                         appslisters.ReplicaSetLister
	statefulSets                        appslisters.StatefulSetLister
	enableMinDomainsInPodTopologySpread bool
}

func (pl PodTopologySpread) Name() string {
	return "PodTopologySpread"
}

func (pl PodTopologySpread) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#9. %s", pl.Name()))
}

type topologySpreadConstraint struct {
	MaxSkew     int32
	TopologyKey string
	Selector    labels.Selector
	MinDomains  int32
}

type criticalPaths [2]struct {
	// TopologyValue denotes the topology value mapping to topology key.
	TopologyValue string
	// MatchNum denotes the number of matching pods.
	MatchNum int
}

// calPreFilterState computes preFilterState describing how pods are spread on topologies.
func (pl *PodTopologySpread) calPreFilterState(pod *v1.Pod, cache *r.NodeCache) (*preFilterState9, error) {

	var constraints []topologySpreadConstraint
	var err error

	if len(pod.Spec.TopologySpreadConstraints) > 0 {
		constraints, err = filterTopologySpreadConstraints(pod.Spec.TopologySpreadConstraints, v1.DoNotSchedule)
		if err != nil {
			return nil, fmt.Errorf("obtaining pod's hard topology spread constraints: %w", err)
		}
	} else {
		constraints, err = pl.buildDefaultConstraints(pod, v1.DoNotSchedule)
		if err != nil {
			return nil, fmt.Errorf("setting default hard topology spread constraints: %w", err)
		}
	}

	if len(constraints) == 0 {
		return &preFilterState9{}, nil
	}

	s := preFilterState9{
		Constraints:          constraints,
		TpKeyToCriticalPaths: make(map[string]*criticalPaths, len(constraints)),
		TpPairToMatchNum:     make(map[topologyPair]int, sizeHeuristic(cache.AvailableNodeCount, constraints)),
	}

	requiredSchedulingTerm := nodeaffinity.GetRequiredNodeAffinity(pod)
	tpCountsByNode := make([]map[topologyPair]int, cache.AvailableNodeCount)
	i := 0
	for _, nodeInfo := range cache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			node := nodeInfo.Node()
			if node == nil {
				klog.ErrorS(nil, "Node not found")
				continue
			}

			match, _ := requiredSchedulingTerm.Match(node)
			if !match {
				continue
			}

			if !nodeLabelsMatchSpreadConstraints(node.Labels, constraints) {
				continue
			}

			tpCounts := make(map[topologyPair]int, len(constraints))
			for _, c := range constraints {
				pair := topologyPair{key: c.TopologyKey, value: node.Labels[c.TopologyKey]}
				count := countPodsMatchSelector(nodeInfo.Pods, c.Selector, pod.Namespace)
				tpCounts[pair] = count
			}
			tpCountsByNode[i] = tpCounts
			i++
		}
	}

	for _, tpCounts := range tpCountsByNode {
		for tp, count := range tpCounts {
			s.TpPairToMatchNum[tp] += count
		}
	}
	if pl.enableMinDomainsInPodTopologySpread {
		s.TpKeyToDomainsNum = make(map[string]int, len(constraints))
		for tp := range s.TpPairToMatchNum {
			s.TpKeyToDomainsNum[tp.key]++
		}
	}

	// calculate min match for each topology pair
	for i := 0; i < len(constraints); i++ {
		key := constraints[i].TopologyKey
		s.TpKeyToCriticalPaths[key] = newCriticalPaths()
	}
	for pair, num := range s.TpPairToMatchNum {
		s.TpKeyToCriticalPaths[pair.key].update(pair.value, num)
	}

	return &s, nil
}

func (pl PodTopologySpread) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	s, err := pl.calPreFilterState(newPod.Pod, nodeInfoCache)

	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if err != nil {
				fmt.Println("calPreFilterState error")
				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
				nodeInfoCache.NodeCountDown()
				newPod.FilterNode(nodeName, pl.Name(), "")
				continue
			}

			// However, "empty" preFilterState is legit which tolerates every toSchedule Pod.
			if len(s.Constraints) == 0 {
				continue
			}

			podLabelSet := labels.Set(newPod.Pod.Labels)
			for _, c := range s.Constraints {
				tpKey := c.TopologyKey
				tpVal, ok := nodeinfo.Node().Labels[c.TopologyKey]
				if !ok {
					fmt.Println("Node doesn't have required label", "node", nodeName, "label", tpKey)
					nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
					nodeInfoCache.NodeCountDown()
					newPod.FilterNode(nodeName, pl.Name(), "")
					continue
				}

				selfMatchNum := 0
				if c.Selector.Matches(podLabelSet) {
					selfMatchNum = 1
				}

				pair := topologyPair{key: tpKey, value: tpVal}

				// judging criteria:
				// 'existing matching num' + 'if self-match (1 or 0)' - 'global minimum' <= 'maxSkew'
				minMatchNum, err := s.minMatchNum(tpKey, c.MinDomains, pl.enableMinDomainsInPodTopologySpread)
				if err != nil {
					fmt.Println(err, "Internal error occurred while retrieving value precalculated in PreFilter", "topologyKey", tpKey, "paths", s.TpKeyToCriticalPaths)
					nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
					nodeInfoCache.NodeCountDown()
					newPod.FilterNode(nodeName, pl.Name(), "")
					continue
				}

				matchNum := 0
				if tpCount, ok := s.TpPairToMatchNum[pair]; ok {
					matchNum = tpCount
				}
				skew := matchNum + selfMatchNum - minMatchNum
				if skew > int(c.MaxSkew) {
					fmt.Println("Node failed spreadConstraint: matchNum + selfMatchNum - minMatchNum > maxSkew", "node", nodeName, "topologyKey", tpKey, "matchNum", matchNum, "selfMatchNum", selfMatchNum, "minMatchNum", minMatchNum, "maxSkew", c.MaxSkew)
					nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
					nodeInfoCache.NodeCountDown()
					newPod.FilterNode(nodeName, pl.Name(), "")
					continue
				}
			}
		}
	}
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

func (pl *PodTopologySpread) buildDefaultConstraints(p *v1.Pod, action v1.UnsatisfiableConstraintAction) ([]topologySpreadConstraint, error) {
	constraints, err := filterTopologySpreadConstraints(pl.defaultConstraints, action)
	if err != nil || len(constraints) == 0 {
		return nil, err
	}
	selector := defaultSelector(p, pl.services, pl.replicationCtrls, pl.replicaSets, pl.statefulSets)
	if selector.Empty() {
		return nil, nil
	}
	for i := range constraints {
		constraints[i].Selector = selector
	}
	return constraints, nil
}

func sizeHeuristic(nodes int, constraints []topologySpreadConstraint) int {
	for _, c := range constraints {
		if c.TopologyKey == v1.LabelHostname {
			return nodes
		}
	}
	return 0
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

func nodeLabelsMatchSpreadConstraints(nodeLabels map[string]string, constraints []topologySpreadConstraint) bool {
	for _, c := range constraints {
		if _, ok := nodeLabels[c.TopologyKey]; !ok {
			return false
		}
	}
	return true
}

func newCriticalPaths() *criticalPaths {
	return &criticalPaths{{MatchNum: math.MaxInt32}, {MatchNum: math.MaxInt32}}
}

func (p *criticalPaths) update(tpVal string, num int) {
	// first verify if `tpVal` exists or not
	i := -1
	if tpVal == p[0].TopologyValue {
		i = 0
	} else if tpVal == p[1].TopologyValue {
		i = 1
	}

	if i >= 0 {
		// `tpVal` exists
		p[i].MatchNum = num
		if p[0].MatchNum > p[1].MatchNum {
			// swap paths[0] and paths[1]
			p[0], p[1] = p[1], p[0]
		}
	} else {
		// `tpVal` doesn't exist
		if num < p[0].MatchNum {
			// update paths[1] with paths[0]
			p[1] = p[0]
			// update paths[0]
			p[0].TopologyValue, p[0].MatchNum = tpVal, num
		} else if num < p[1].MatchNum {
			// update paths[1]
			p[1].TopologyValue, p[1].MatchNum = tpVal, num
		}
	}
}

// minMatchNum returns the global minimum for the calculation of skew while taking MinDomains into account.
func (s *preFilterState9) minMatchNum(tpKey string, minDomains int32, enableMinDomainsInPodTopologySpread bool) (int, error) {
	paths, ok := s.TpKeyToCriticalPaths[tpKey]
	if !ok {
		return 0, fmt.Errorf("failed to retrieve path by topology key")
	}

	minMatchNum := paths[0].MatchNum
	if !enableMinDomainsInPodTopologySpread {
		return minMatchNum, nil
	}

	domainsNum, ok := s.TpKeyToDomainsNum[tpKey]
	if !ok {
		return 0, fmt.Errorf("failed to retrieve the number of domains by topology key")
	}

	if domainsNum < int(minDomains) {
		// When the number of eligible domains with matching topology keys is less than `minDomains`,
		// it treats "global minimum" as 0.
		minMatchNum = 0
	}

	return minMatchNum, nil
}

var (
	rcKind = v1.SchemeGroupVersion.WithKind("ReplicationController")
	rsKind = appsv1.SchemeGroupVersion.WithKind("ReplicaSet")
	ssKind = appsv1.SchemeGroupVersion.WithKind("StatefulSet")
)

func defaultSelector(
	pod *v1.Pod,
	sl corelisters.ServiceLister,
	cl corelisters.ReplicationControllerLister,
	rsl appslisters.ReplicaSetLister,
	ssl appslisters.StatefulSetLister,
) labels.Selector {
	labelSet := make(labels.Set)
	// Since services, RCs, RSs and SSs match the pod, they won't have conflicting
	// labels. Merging is safe.

	if services, err := GetPodServices(sl, pod); err == nil {
		for _, service := range services {
			labelSet = labels.Merge(labelSet, service.Spec.Selector)
		}
	}
	selector := labelSet.AsSelector()

	owner := metav1.GetControllerOfNoCopy(pod)
	if owner == nil {
		return selector
	}

	gv, err := schema.ParseGroupVersion(owner.APIVersion)
	if err != nil {
		return selector
	}

	gvk := gv.WithKind(owner.Kind)
	switch gvk {
	case rcKind:
		if rc, err := cl.ReplicationControllers(pod.Namespace).Get(owner.Name); err == nil {
			labelSet = labels.Merge(labelSet, rc.Spec.Selector)
			selector = labelSet.AsSelector()
		}
	case rsKind:
		if rs, err := rsl.ReplicaSets(pod.Namespace).Get(owner.Name); err == nil {
			if other, err := metav1.LabelSelectorAsSelector(rs.Spec.Selector); err == nil {
				if r, ok := other.Requirements(); ok {
					selector = selector.Add(r...)
				}
			}
		}
	case ssKind:
		if ss, err := ssl.StatefulSets(pod.Namespace).Get(owner.Name); err == nil {
			if other, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector); err == nil {
				if r, ok := other.Requirements(); ok {
					selector = selector.Add(r...)
				}
			}
		}
	default:
		// Not owned by a supported controller.
	}

	return selector
}

// GetPodServices gets the services that have the selector that match the labels on the given pod.
func GetPodServices(sl corelisters.ServiceLister, pod *v1.Pod) ([]*v1.Service, error) {
	allServices, err := sl.Services(pod.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var services []*v1.Service
	for i := range allServices {
		service := allServices[i]
		if service.Spec.Selector == nil {
			// services with nil selectors match nothing, not everything.
			continue
		}
		selector := labels.Set(service.Spec.Selector).AsSelectorPreValidated()
		if selector.Matches(labels.Set(pod.Labels)) {
			services = append(services, service)
		}
	}

	return services, nil
}
