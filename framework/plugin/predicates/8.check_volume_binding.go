package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type CheckVolumeBinding struct{}

// ConflictReason is used for the special strings which explain why
// volume binding is impossible for a node.
type ConflictReason string

// ConflictReasons contains all reasons that explain why volume binding is impossible for a node.
type ConflictReasons []ConflictReason

const (
	// ErrReasonBindConflict is used for VolumeBindingNoMatch predicate error.
	ErrReasonBindConflict ConflictReason = "node(s) didn't find available persistent volumes to bind"
	// ErrReasonNodeConflict is used for VolumeNodeAffinityConflict predicate error.
	ErrReasonNodeConflict ConflictReason = "node(s) had volume node affinity conflict"
	// ErrReasonNotEnoughSpace is used when a pod cannot start on a node because not enough storage space is available.
	ErrReasonNotEnoughSpace = "node(s) did not have enough free storage"
	// ErrReasonPVNotExist is used when a pod has one or more PVC(s) bound to non-existent persistent volume(s)"
	ErrReasonPVNotExist = "node(s) unavailable due to one or more pvc(s) bound to non-existent pv(s)"
)

// func (pl CheckVolumeBinding) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
// 	reason, err, skip, stateData := pl.PreFilter(newPod.Pod)

// 	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
// 		if !nodeinfo.PluginResult.IsFiltered {
// 			podVolumes, reasons, err := FindPodVolumes(newPod.Pod, stateData.boundClaims, stateData.claimsToBind, nodeinfo)
// 		}
// 	}
// }

func (pl CheckVolumeBinding) Name() string {
	return "CheckVolumeBinding"
}

func (pl CheckVolumeBinding) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#1. %s", pl.Name()))
}

func (pl CheckVolumeBinding) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			//filter
		}
	}
}

// // FindPodVolumes finds the matching PVs for PVCs and nodes to provision PVs
// // for the given pod and node. If the node does not fit, confilict reasons are
// // returned.
// func FindPodVolumes(pod *v1.Pod, boundClaims, claimsToBind []*v1.PersistentVolumeClaim, nodeInfo *r.NodeInfo) (podVolumes *r.PodVolumes, reasons ConflictReasons, err error) {
// 	podVolumes = &r.PodVolumes{}

// 	// Warning: Below log needs high verbosity as it can be printed several times (#60933).
// 	klog.V(5).InfoS("FindPodVolumes", "pod", klog.KObj(pod), "node", klog.KObj(nodeInfo.Node()))

// 	// Initialize to true for pods that don't have volumes. These
// 	// booleans get translated into reason strings when the function
// 	// returns without an error.
// 	unboundVolumesSatisfied := true
// 	boundVolumesSatisfied := true
// 	sufficientStorage := true
// 	boundPVsFound := true
// 	defer func() {
// 		if err != nil {
// 			return
// 		}
// 		if !boundVolumesSatisfied {
// 			reasons = append(reasons, ErrReasonNodeConflict)
// 		}
// 		if !unboundVolumesSatisfied {
// 			reasons = append(reasons, ErrReasonBindConflict)
// 		}
// 		if !sufficientStorage {
// 			reasons = append(reasons, ErrReasonNotEnoughSpace)
// 		}
// 		if !boundPVsFound {
// 			reasons = append(reasons, ErrReasonPVNotExist)
// 		}
// 	}()

// 	var (
// 		staticBindings    []*r.BindingInfo
// 		dynamicProvisions []*v1.PersistentVolumeClaim
// 	)
// 	defer func() {
// 		// Although we do not distinguish nil from empty in this function, for
// 		// easier testing, we normalize empty to nil.
// 		if len(staticBindings) == 0 {
// 			staticBindings = nil
// 		}
// 		if len(dynamicProvisions) == 0 {
// 			dynamicProvisions = nil
// 		}
// 		podVolumes.StaticBindings = staticBindings
// 		podVolumes.DynamicProvisions = dynamicProvisions
// 	}()

// 	// Check PV node affinity on bound volumes
// 	if len(boundClaims) > 0 {
// 		boundVolumesSatisfied, boundPVsFound, err = b.checkBoundClaims(boundClaims, node, pod)
// 		if err != nil {
// 			return
// 		}
// 	}

// 	// Find matching volumes and node for unbound claims
// 	if len(claimsToBind) > 0 {
// 		var (
// 			claimsToFindMatching []*v1.PersistentVolumeClaim
// 			claimsToProvision    []*v1.PersistentVolumeClaim
// 		)

// 		// Filter out claims to provision
// 		for _, claim := range claimsToBind {
// 			if selectedNode, ok := claim.Annotations[volume.AnnSelectedNode]; ok {
// 				if selectedNode != node.Name {
// 					// Fast path, skip unmatched node.
// 					unboundVolumesSatisfied = false
// 					return
// 				}
// 				claimsToProvision = append(claimsToProvision, claim)
// 			} else {
// 				claimsToFindMatching = append(claimsToFindMatching, claim)
// 			}
// 		}

// 		// Find matching volumes
// 		if len(claimsToFindMatching) > 0 {
// 			var unboundClaims []*v1.PersistentVolumeClaim
// 			unboundVolumesSatisfied, staticBindings, unboundClaims, err = b.findMatchingVolumes(pod, claimsToFindMatching, node)
// 			if err != nil {
// 				return
// 			}
// 			claimsToProvision = append(claimsToProvision, unboundClaims...)
// 		}

// 		// Check for claims to provision. This is the first time where we potentially
// 		// find out that storage is not sufficient for the node.
// 		if len(claimsToProvision) > 0 {
// 			unboundVolumesSatisfied, sufficientStorage, dynamicProvisions, err = b.checkVolumeProvisions(pod, claimsToProvision, node)
// 			if err != nil {
// 				return
// 			}
// 		}
// 	}

// 	return
// }
