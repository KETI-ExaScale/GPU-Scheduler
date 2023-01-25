package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
	"log"
	"sort"
	"sync"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/component-helpers/storage/ephemeral"
	"k8s.io/component-helpers/storage/volume"
	"k8s.io/klog/v2"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
)

type CheckVolumeBinding struct {
	PVCLister   corelisters.PersistentVolumeClaimLister
	ClassLister storagelisters.StorageClassLister
	PVCCache    PVCAssumeCache
}

// PodVolumes holds pod's volumes information used in volume scheduling.
type PodVolumes struct {
	// StaticBindings are binding decisions for PVCs which can be bound to
	// pre-provisioned static PVs.
	StaticBindings []*BindingInfo
	// DynamicProvisions are PVCs that require dynamic provisioning
	DynamicProvisions []*v1.PersistentVolumeClaim
}

type BindingInfo struct {
	// PVC that needs to be bound
	pvc *v1.PersistentVolumeClaim

	// Proposed PV to bind to this PVC
	pv *v1.PersistentVolume
}

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

type stateData struct {
	skip         bool // set true if pod does not have PVCs
	boundClaims  []*v1.PersistentVolumeClaim
	claimsToBind []*v1.PersistentVolumeClaim
	allBound     bool
	// podVolumesByNode holds the pod's volume information found in the Filter
	// phase for each node
	// it's initialized in the PreFilter phase
	podVolumesByNode map[string]*PodVolumes
	sync.Mutex
}

func (pl CheckVolumeBinding) Name() string {
	return "CheckVolumeBinding"
}

func (pl CheckVolumeBinding) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#8. %s", pl.Name()))
}

// PreFilter invoked at the prefilter extension point to check if pod has all
// immediate PVCs bound. If not all immediate PVCs are bound, an
// UnschedulableAndUnresolvable is returned.
func (pl CheckVolumeBinding) PreFilter(pod *v1.Pod) (string, error, bool, *stateData) {
	// If pod does not reference any PVC, we don't need to do anything.
	if hasPVC, err := pl.podHasPVCs(pod); err != nil {
		return "podHasPVC error", err, true, nil
	} else if !hasPVC {
		return "", nil, true, nil
	}
	boundClaims, claimsToBind, unboundClaimsImmediate, err := pl.getPodVolumes(pod)
	if err != nil {
		return "getPodVolumes error", err, true, nil
	}
	if len(unboundClaimsImmediate) > 0 {
		// Return UnschedulableAndUnresolvable error if immediate claims are
		// not bound. Pod will be moved to active/backoff queues once these
		// claims are bound by PV controller.
		return "pod has unbound immediate PersistentVolumeClaims", nil, true, nil
	}
	stateData_ := &stateData{boundClaims: boundClaims, claimsToBind: claimsToBind, podVolumesByNode: make(map[string]*PodVolumes)}
	return "", nil, false, stateData_
}

func (pl CheckVolumeBinding) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	reason, err, skip, stateData := pl.PreFilter(newPod.Pod)

	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if skip {
				if err != nil {
					filterState := r.FilterStatus{r.Error, pl.Name(), reason, err}
					nodeinfo.PluginResult.FilterNode(nodeName, filterState)
					nodeInfoCache.NodeCountDown()
				} else {
					filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
					nodeinfo.PluginResult.FilterNode(nodeName, filterState)
					nodeInfoCache.NodeCountDown()
				}
				return
			}

			_, reasons, err := pl.findPodVolumes(newPod.Pod, stateData.boundClaims, stateData.claimsToBind, nodeinfo.Node()) //스코어링 준비

			if err != nil {
				filterState := r.FilterStatus{r.Error, pl.Name(), "findPodVolumes error", err}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				continue
			}

			if len(reasons) > 0 {
				var reasons string
				for _, reason := range reasons {
					reasons += string(reason) + "/"
				}
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reasons, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				continue
			}
		}
	}
}

// podHasPVCs returns 2 values:
// - the first one to denote if the given "pod" has any PVC defined.
// - the second one to return any error if the requested PVC is illegal.
func (pl *CheckVolumeBinding) podHasPVCs(pod *v1.Pod) (bool, error) {
	hasPVC := false
	for _, vol := range pod.Spec.Volumes {
		var pvcName string
		isEphemeral := false
		switch {
		case vol.PersistentVolumeClaim != nil:
			pvcName = vol.PersistentVolumeClaim.ClaimName
		case vol.Ephemeral != nil:
			pvcName = ephemeral.VolumeClaimName(pod, &vol)
			isEphemeral = true
		default:
			// Volume is not using a PVC, ignore
			continue
		}
		hasPVC = true
		pvc, err := pl.PVCLister.PersistentVolumeClaims(pod.Namespace).Get(pvcName)
		if err != nil {
			// The error usually has already enough context ("persistentvolumeclaim "myclaim" not found"),
			// but we can do better for generic ephemeral inline volumes where that situation
			// is normal directly after creating a pod.
			if isEphemeral && apierrors.IsNotFound(err) {
				err = fmt.Errorf("waiting for ephemeral volume controller to create the persistentvolumeclaim %q", pvcName)
			}
			return hasPVC, err
		}

		if pvc.Status.Phase == v1.ClaimLost {
			return hasPVC, fmt.Errorf("persistentvolumeclaim %q bound to non-existent persistentvolume %q", pvc.Name, pvc.Spec.VolumeName)
		}

		if pvc.DeletionTimestamp != nil {
			return hasPVC, fmt.Errorf("persistentvolumeclaim %q is being deleted", pvc.Name)
		}

		if isEphemeral {
			if err := ephemeral.VolumeIsForPod(pod, pvc); err != nil {
				return hasPVC, err
			}
		}
	}
	return hasPVC, nil
}

func CheckVolumeBinding_() CheckVolumeBinding {
	hostConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)
	informerFactory := informers.NewSharedInformerFactory(hostKubeClient, 0)
	pvcLister := informerFactory.Core().V1().PersistentVolumeClaims().Lister()
	classLister := informerFactory.Storage().V1().StorageClasses().Lister()

	return CheckVolumeBinding{
		PVCLister:   pvcLister,
		ClassLister: classLister,
	}
}

// GetPodVolumes returns a pod's PVCs separated into bound, unbound with delayed binding (including provisioning)
// and unbound with immediate binding (including prebound)
func (pl CheckVolumeBinding) getPodVolumes(pod *v1.Pod) (boundClaims []*v1.PersistentVolumeClaim, unboundClaimsDelayBinding []*v1.PersistentVolumeClaim, unboundClaimsImmediate []*v1.PersistentVolumeClaim, err error) {
	boundClaims = []*v1.PersistentVolumeClaim{}
	unboundClaimsImmediate = []*v1.PersistentVolumeClaim{}
	unboundClaimsDelayBinding = []*v1.PersistentVolumeClaim{}

	for _, vol := range pod.Spec.Volumes {
		volumeBound, pvc, err := isVolumeBound(pod, &vol)
		if err != nil {
			return nil, nil, nil, err
		}
		if pvc == nil {
			continue
		}
		if volumeBound {
			boundClaims = append(boundClaims, pvc)
		} else {
			delayBindingMode, err := volume.IsDelayBindingMode(pvc, pl.ClassLister)
			if err != nil {
				return nil, nil, nil, err
			}
			// Prebound PVCs are treated as unbound immediate binding
			if delayBindingMode && pvc.Spec.VolumeName == "" {
				// Scheduler path
				unboundClaimsDelayBinding = append(unboundClaimsDelayBinding, pvc)
			} else {
				// !delayBindingMode || pvc.Spec.VolumeName != ""
				// Immediate binding should have already been bound
				unboundClaimsImmediate = append(unboundClaimsImmediate, pvc)
			}
		}
	}
	return boundClaims, unboundClaimsDelayBinding, unboundClaimsImmediate, nil
}

func isVolumeBound(pod *v1.Pod, vol *v1.Volume) (bound bool, pvc *v1.PersistentVolumeClaim, err error) {
	pvcName := ""
	isEphemeral := false
	switch {
	case vol.PersistentVolumeClaim != nil:
		pvcName = vol.PersistentVolumeClaim.ClaimName
	case vol.Ephemeral != nil:
		// Generic ephemeral inline volumes also use a PVC,
		// just with a computed name, and...
		pvcName = ephemeral.VolumeClaimName(pod, vol)
		isEphemeral = true
	default:
		return true, nil, nil
	}

	bound, pvc, err = isPVCBound(pod.Namespace, pvcName)
	// ... the PVC must be owned by the pod.
	if isEphemeral && err == nil && pvc != nil {
		if err := ephemeral.VolumeIsForPod(pod, pvc); err != nil {
			return false, nil, err
		}
	}
	return
}

func (pl CheckVolumeBinding) isPVCBound(namespace, pvcName string) (bool, *v1.PersistentVolumeClaim, error) {
	claim := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
	}
	pvcKey := getPVCName(claim)
	pvc, err := b.pvcCache.GetPVC(pvcKey) //일단스킵
	if err != nil || pvc == nil {
		return false, nil, fmt.Errorf("error getting PVC %q: %v", pvcKey, err)
	}

	fullyBound := isPVCFullyBound(pvc)
	if fullyBound {
		klog.V(5).InfoS("PVC is fully bound to PV", "PVC", klog.KObj(pvc), "PV", klog.KRef("", pvc.Spec.VolumeName))
	} else {
		if pvc.Spec.VolumeName != "" {
			klog.V(5).InfoS("PVC is not fully bound to PV", "PVC", klog.KObj(pvc), "PV", klog.KRef("", pvc.Spec.VolumeName))
		} else {
			klog.V(5).InfoS("PVC is not bound", "PVC", klog.KObj(pvc))
		}
	}
	return fullyBound, pvc, nil
}

func getPVCName(pvc *v1.PersistentVolumeClaim) string {
	return pvc.Namespace + "/" + pvc.Name
}

func isPVCFullyBound(pvc *v1.PersistentVolumeClaim) bool {
	return pvc.Spec.VolumeName != "" && metav1.HasAnnotation(pvc.ObjectMeta, volume.AnnBindCompleted)
}

// NewPVCAssumeCache creates a PVC assume cache.
func NewPVCAssumeCache(informer cache.SharedIndexInformer) PVCAssumeCache {
	return &pvcAssumeCache{NewAssumeCache(informer, "v1.PersistentVolumeClaim", "", nil)}
}

// NewAssumeCache creates an assume cache for general objects.
func NewAssumeCache(informer cache.SharedIndexInformer, description, indexName string, indexFunc cache.IndexFunc) AssumeCache {
	c := &assumeCache{
		description: description,
		indexFunc:   indexFunc,
		indexName:   indexName,
	}
	indexers := cache.Indexers{}
	if indexName != "" && indexFunc != nil {
		indexers[indexName] = c.objInfoIndexFunc
	}
	c.store = cache.NewIndexer(objInfoKeyFunc, indexers)

	// Unit tests don't use informers
	if informer != nil {
		informer.AddEventHandler(
			cache.ResourceEventHandlerFuncs{
				AddFunc:    c.add,
				UpdateFunc: c.update,
				DeleteFunc: c.delete,
			},
		)
	}
	return c
}

// FindPodVolumes finds the matching PVs for PVCs and nodes to provision PVs
// for the given pod and node. If the node does not fit, confilict reasons are
// returned.
func (pl CheckVolumeBinding) findPodVolumes(pod *v1.Pod, boundClaims, claimsToBind []*v1.PersistentVolumeClaim, node *v1.Node) (podVolumes *PodVolumes, reasons ConflictReasons, err error) {
	podVolumes = &PodVolumes{}

	// Warning: Below log needs high verbosity as it can be printed several times (#60933).
	klog.V(5).InfoS("FindPodVolumes", "pod", klog.KObj(pod), "node", klog.KObj(node))

	// Initialize to true for pods that don't have volumes. These
	// booleans get translated into reason strings when the function
	// returns without an error.
	unboundVolumesSatisfied := true
	boundVolumesSatisfied := true
	sufficientStorage := true
	boundPVsFound := true
	defer func() {
		if err != nil {
			return
		}
		if !boundVolumesSatisfied {
			reasons = append(reasons, ErrReasonNodeConflict)
		}
		if !unboundVolumesSatisfied {
			reasons = append(reasons, ErrReasonBindConflict)
		}
		if !sufficientStorage {
			reasons = append(reasons, ErrReasonNotEnoughSpace)
		}
		if !boundPVsFound {
			reasons = append(reasons, ErrReasonPVNotExist)
		}
	}()

	defer func() {
		if err != nil {
			// metrics.VolumeSchedulingStageFailed.WithLabelValues("predicate").Inc()
			// 에러기록
		}
	}()

	var (
		staticBindings    []*BindingInfo
		dynamicProvisions []*v1.PersistentVolumeClaim
	)
	defer func() {
		// Although we do not distinguish nil from empty in this function, for
		// easier testing, we normalize empty to nil.
		if len(staticBindings) == 0 {
			staticBindings = nil
		}
		if len(dynamicProvisions) == 0 {
			dynamicProvisions = nil
		}
		podVolumes.StaticBindings = staticBindings
		podVolumes.DynamicProvisions = dynamicProvisions
	}()

	// Check PV node affinity on bound volumes
	if len(boundClaims) > 0 {
		boundVolumesSatisfied, boundPVsFound, err = checkBoundClaims(boundClaims, node, pod)
		if err != nil {
			return
		}
	}

	// Find matching volumes and node for unbound claims
	if len(claimsToBind) > 0 {
		var (
			claimsToFindMatching []*v1.PersistentVolumeClaim
			claimsToProvision    []*v1.PersistentVolumeClaim
		)

		// Filter out claims to provision
		for _, claim := range claimsToBind {
			if selectedNode, ok := claim.Annotations[volume.AnnSelectedNode]; ok {
				if selectedNode != node.Name {
					// Fast path, skip unmatched node.
					unboundVolumesSatisfied = false
					return
				}
				claimsToProvision = append(claimsToProvision, claim)
			} else {
				claimsToFindMatching = append(claimsToFindMatching, claim)
			}
		}

		// Find matching volumes
		if len(claimsToFindMatching) > 0 {
			var unboundClaims []*v1.PersistentVolumeClaim
			unboundVolumesSatisfied, staticBindings, unboundClaims, err = findMatchingVolumes(pod, claimsToFindMatching, node)
			if err != nil {
				return
			}
			claimsToProvision = append(claimsToProvision, unboundClaims...)
		}

		// Check for claims to provision. This is the first time where we potentially
		// find out that storage is not sufficient for the node.
		if len(claimsToProvision) > 0 {
			unboundVolumesSatisfied, sufficientStorage, dynamicProvisions, err = pl.checkVolumeProvisions(pod, claimsToProvision, node)
			if err != nil {
				return
			}
		}
	}

	return
}

// findMatchingVolumes tries to find matching volumes for given claims,
// and return unbound claims for further provision.
func (pl CheckVolumeBinding) findMatchingVolumes(pod *v1.Pod, claimsToBind []*v1.PersistentVolumeClaim, node *v1.Node) (foundMatches bool, bindings []*BindingInfo, unboundClaims []*v1.PersistentVolumeClaim, err error) {
	// Sort all the claims by increasing size request to get the smallest fits
	sort.Sort(byPVCSize(claimsToBind))

	chosenPVs := map[string]*v1.PersistentVolume{}

	foundMatches = true

	for _, pvc := range claimsToBind {
		// Get storage class name from each PVC
		storageClassName := volume.GetPersistentVolumeClaimClass(pvc)
		allPVs := b.pvCache.ListPVs(storageClassName) //일단스킵

		// Find a matching PV
		pv, err := volume.FindMatchingVolume(pvc, allPVs, node, chosenPVs, true)
		if err != nil {
			return false, nil, nil, err
		}
		if pv == nil {
			klog.V(4).InfoS("No matching volumes for pod", "pod", klog.KObj(pod), "PVC", klog.KObj(pvc), "node", klog.KObj(node))
			unboundClaims = append(unboundClaims, pvc)
			foundMatches = false
			continue
		}

		// matching PV needs to be excluded so we don't select it again
		chosenPVs[pv.Name] = pv
		bindings = append(bindings, &BindingInfo{pv: pv, pvc: pvc})
		klog.V(5).InfoS("Found matching PV for PVC for pod", "PV", klog.KObj(pv), "PVC", klog.KObj(pvc), "node", klog.KObj(node), "pod", klog.KObj(pod))
	}

	if foundMatches {
		klog.V(4).InfoS("Found matching volumes for pod", "pod", klog.KObj(pod), "node", klog.KObj(node))
	}

	return
}

type byPVCSize []*v1.PersistentVolumeClaim

func (a byPVCSize) Len() int {
	return len(a)
}

func (a byPVCSize) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a byPVCSize) Less(i, j int) bool {
	iSize := a[i].Spec.Resources.Requests[v1.ResourceStorage]
	jSize := a[j].Spec.Resources.Requests[v1.ResourceStorage]
	// return true if iSize is less than jSize
	return iSize.Cmp(jSize) == -1
}

// checkVolumeProvisions checks given unbound claims (the claims have gone through func
// findMatchingVolumes, and do not have matching volumes for binding), and return true
// if all of the claims are eligible for dynamic provision.
func (pl CheckVolumeBinding) checkVolumeProvisions(pod *v1.Pod, claimsToProvision []*v1.PersistentVolumeClaim, node *v1.Node) (provisionSatisfied, sufficientStorage bool, dynamicProvisions []*v1.PersistentVolumeClaim, err error) {
	dynamicProvisions = []*v1.PersistentVolumeClaim{}

	// We return early with provisionedClaims == nil if a check
	// fails or we encounter an error.
	for _, claim := range claimsToProvision {
		pvcName := getPVCName(claim)
		className := volume.GetPersistentVolumeClaimClass(claim)
		if className == "" {
			return false, false, nil, fmt.Errorf("no class for claim %q", pvcName)
		}

		class, err := pl.ClassLister.Get(className)
		if err != nil {
			return false, false, nil, fmt.Errorf("failed to find storage class %q", className)
		}
		provisioner := class.Provisioner
		if provisioner == "" || provisioner == volume.NotSupportedProvisioner {
			klog.V(4).InfoS("Storage class of claim does not support dynamic provisioning", "storageClassName", className, "PVC", klog.KObj(claim))
			return false, true, nil, nil
		}

		// Check if the node can satisfy the topology requirement in the class
		if !v1helper.MatchTopologySelectorTerms(class.AllowedTopologies, labels.Set(node.Labels)) {
			klog.V(4).InfoS("Node cannot satisfy provisioning topology requirements of claim", "node", klog.KObj(node), "PVC", klog.KObj(claim))
			return false, true, nil, nil
		}

		// Check storage capacity.
		sufficient, err := hasEnoughCapacity(provisioner, claim, class, node)
		if err != nil {
			return false, false, nil, err
		}
		if !sufficient {
			// hasEnoughCapacity logs an explanation.
			return true, false, nil, nil
		}

		dynamicProvisions = append(dynamicProvisions, claim)

	}
	klog.V(4).InfoS("Provisioning for claims of pod that has no matching volumes...", "claimCount", len(claimsToProvision), "pod", klog.KObj(pod), "node", klog.KObj(node))

	return true, true, dynamicProvisions, nil
}

// hasEnoughCapacity checks whether the provisioner has enough capacity left for a new volume of the given size
// that is available from the node.
func (pl CheckVolumeBinding) hasEnoughCapacity(provisioner string, claim *v1.PersistentVolumeClaim, storageClass *storagev1.StorageClass, node *v1.Node) (bool, error) {
	quantity, ok := claim.Spec.Resources.Requests[v1.ResourceStorage]
	if !ok {
		// No capacity to check for.
		return true, nil
	}

	// Only enabled for CSI drivers which opt into it.
	driver, err := b.csiDriverLister.Get(provisioner)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Either the provisioner is not a CSI driver or the driver does not
			// opt into storage capacity scheduling. Either way, skip
			// capacity checking.
			return true, nil
		}
		return false, err
	}
	if driver.Spec.StorageCapacity == nil || !*driver.Spec.StorageCapacity {
		return true, nil
	}

	// Look for a matching CSIStorageCapacity object(s).
	// TODO (for beta): benchmark this and potentially introduce some kind of lookup structure (https://github.com/kubernetes/enhancements/issues/1698#issuecomment-654356718).
	capacities, err := b.csiStorageCapacityLister.List(labels.Everything())
	if err != nil {
		return false, err
	}

	sizeInBytes := quantity.Value()
	for _, capacity := range capacities {
		if capacity.StorageClassName == storageClass.Name &&
			capacitySufficient(capacity, sizeInBytes) &&
			nodeHasAccess(node, capacity) {
			// Enough capacity found.
			return true, nil
		}
	}

	// TODO (?): this doesn't give any information about which pools where considered and why
	// they had to be rejected. Log that above? But that might be a lot of log output...
	klog.V(4).InfoS("Node has no accessible CSIStorageCapacity with enough capacity for PVC",
		"node", klog.KObj(node), "PVC", klog.KObj(claim), "size", sizeInBytes, "storageClass", klog.KObj(storageClass))
	return false, nil
}

func (pl CheckVolumeBinding) capacitySufficient(capacity *storagev1.CSIStorageCapacity, sizeInBytes int64) bool {
	limit := capacity.Capacity
	if capacity.MaximumVolumeSize != nil {
		// Prefer MaximumVolumeSize if available, it is more precise.
		limit = capacity.MaximumVolumeSize
	}
	return limit != nil && limit.Value() >= sizeInBytes
}

func (pl CheckVolumeBinding) nodeHasAccess(node *v1.Node, capacity *storagev1.CSIStorageCapacity) bool {
	if capacity.NodeTopology == nil {
		// Unavailable
		return false
	}
	// Only matching by label is supported.
	selector, err := metav1.LabelSelectorAsSelector(capacity.NodeTopology)
	if err != nil {
		klog.ErrorS(err, "Unexpected error converting to a label selector", "nodeTopology", capacity.NodeTopology)
		return false
	}
	return selector.Matches(labels.Set(node.Labels))
}

func (pl CheckVolumeBinding) checkBoundClaims(claims []*v1.PersistentVolumeClaim, node *v1.Node, pod *v1.Pod) (bool, bool, error) {
	csiNode, err := b.csiNodeLister.Get(node.Name)
	if err != nil {
		// TODO: return the error once CSINode is created by default
		klog.V(4).InfoS("Could not get a CSINode object for the node", "node", klog.KObj(node), "err", err)
	}

	for _, pvc := range claims {
		pvName := pvc.Spec.VolumeName
		pv, err := b.pvCache.GetPV(pvName)
		if err != nil {
			if _, ok := err.(*errNotFound); ok {
				err = nil
			}
			return true, false, err
		}

		pv, err = b.tryTranslatePVToCSI(pv, csiNode)
		if err != nil {
			return false, true, err
		}

		err = volume.CheckNodeAffinity(pv, node.Labels)
		if err != nil {
			klog.V(4).InfoS("PersistentVolume and node mismatch for pod", "PV", klog.KRef("", pvName), "node", klog.KObj(node), "pod", klog.KObj(pod), "err", err)
			return false, true, nil
		}
		klog.V(5).InfoS("PersistentVolume and node matches for pod", "PV", klog.KRef("", pvName), "node", klog.KObj(node), "pod", klog.KObj(pod))
	}

	klog.V(4).InfoS("All bound volumes for pod match with node", "pod", klog.KObj(pod), "node", klog.KObj(node))
	return true, true, nil
}
