package priorities

import (
	r "gpu-scheduler/resourceinfo"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
)

type preScoreState1 struct {
	preferredNodeAffinity *nodeaffinity.PreferredSchedulingTerms
}

type preScoreState2 struct {
	tolerationsPreferNoSchedule []v1.Toleration
}

type preScoreState3 struct {
	selector labels.Selector
}

type preScoreState4 struct {
	topologyScore scoreMap
	podInfo       *r.PodInfo
	// A copy of the incoming pod's namespace labels.
	namespaceLabels labels.Set
}
type scoreMap map[string]map[string]int64

type UnsatisfiableConstraintAction string

type preScoreState5 struct {
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

func DefaultNormalizeScore(maxPriority int64, reverse bool, nodeinfolist []*r.NodeInfo) bool {
	var maxCount int64
	for _, nodeinfo := range nodeinfolist {
		if !nodeinfo.PluginResult.IsFiltered {
			if int64(nodeinfo.PluginResult.NodeScore) > maxCount {
				maxCount = int64(nodeinfo.PluginResult.NodeScore)
			}
		}
	}

	if maxCount == 0 {
		if reverse {
			for _, nodeinfo := range nodeinfolist {
				if !nodeinfo.PluginResult.IsFiltered {
					nodeinfo.PluginResult.NodeScore = int(maxPriority)
				}
			}
		}
		return true
	}

	for _, nodeinfo := range nodeinfolist {
		if !nodeinfo.PluginResult.IsFiltered {
			score := int64(nodeinfo.PluginResult.NodeScore)
			score = maxPriority * score / maxCount
			if reverse {
				score = maxPriority - score
			}
			nodeinfo.PluginResult.NodeScore = int(score)

		}
	}

	return true
}

func DefaultSelector(
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
