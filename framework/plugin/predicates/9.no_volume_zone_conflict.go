package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
	"log"

	corev1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/rest"
	volumehelpers "k8s.io/cloud-provider/volume/helpers"
	storagehelpers "k8s.io/component-helpers/storage/volume"
)

type NoVolumeZoneConflict struct {
	pvLister  corelisters.PersistentVolumeLister
	pvcLister corelisters.PersistentVolumeClaimLister
	scLister  storagelisters.StorageClassLister
}

var volumeZoneLabels = sets.NewString(
	corev1.LabelFailureDomainBetaZone,
	corev1.LabelFailureDomainBetaRegion,
	corev1.LabelTopologyZone,
	corev1.LabelTopologyRegion,
)

const (
	// ErrReasonConflict is used for NoVolumeZoneConflict predicate error.
	ErrReasonConflict = "node(s) had no available volume zone"
)

func (pl NoVolumeZoneConflict) Name() string {
	return "NoVolumeZoneConflict"
}

func (pl NoVolumeZoneConflict) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("[stage] F#9. %s", pl.Name()))
}

func (pl NoVolumeZoneConflict) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	// If a pod doesn't have any volume attached to it, the predicate will always be true.
	// Thus we make a fast path for it, to avoid unnecessary computations in this case.
	if len(newPod.Pod.Spec.Volumes) == 0 {
		return
	}

	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {

			node := nodeinfo.Node()
			nodeConstraints := make(map[string]string)

			for k, v := range node.ObjectMeta.Labels {
				if !volumeZoneLabels.Has(k) {
					continue
				}
				nodeConstraints[k] = v
			}

			if len(nodeConstraints) == 0 {
				continue
			}

			for i := range newPod.Pod.Spec.Volumes {
				volume := newPod.Pod.Spec.Volumes[i]
				if volume.PersistentVolumeClaim == nil {
					continue
				}

				pvcName := volume.PersistentVolumeClaim.ClaimName
				if pvcName == "" {
					reason := "PersistentVolumeClaim had no name"
					filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
					nodeinfo.PluginResult.FilterNode(nodeName, filterState)
					nodeInfoCache.NodeCountDown()
					break
				}

				pvc, err := pl.pvcLister.PersistentVolumeClaims(newPod.Pod.Namespace).Get(pvcName)
				if err != nil {
					reason := "pvclist error"
					filterState := r.FilterStatus{r.Error, pl.Name(), reason, err}
					nodeinfo.PluginResult.FilterNode(nodeName, filterState)
					nodeInfoCache.NodeCountDown()
					break
				}

				pvName := pvc.Spec.VolumeName
				if pvName == "" {
					scName := storagehelpers.GetPersistentVolumeClaimClass(pvc)
					if len(scName) == 0 {
						reason := "PersistentVolumeClaim had no pv name and storageClass name"
						filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
						nodeinfo.PluginResult.FilterNode(nodeName, filterState)
						nodeInfoCache.NodeCountDown()
						break
					}

					class, err := pl.scLister.Get(scName)
					if err != nil {
						reason := "scLister error"
						filterState := r.FilterStatus{r.Error, pl.Name(), reason, err}
						nodeinfo.PluginResult.FilterNode(nodeName, filterState)
						nodeInfoCache.NodeCountDown()
						break
					}

					if class.VolumeBindingMode == nil {
						reason := fmt.Sprintf("VolumeBindingMode not set for StorageClass %q", scName)
						filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
						nodeinfo.PluginResult.FilterNode(nodeName, filterState)
						nodeInfoCache.NodeCountDown()
						break
					}

					if *class.VolumeBindingMode == storage.VolumeBindingWaitForFirstConsumer {
						// Skip unbound volumes
						continue
					}

					reason := "PersistentVolume had no name"
					filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
					nodeinfo.PluginResult.FilterNode(nodeName, filterState)
					nodeInfoCache.NodeCountDown()
					break
				}

				pv, err := pl.pvLister.Get(pvName)
				if err != nil {
					reason := "pvLister error"
					filterState := r.FilterStatus{r.Error, pl.Name(), reason, err}
					nodeinfo.PluginResult.FilterNode(nodeName, filterState)
					nodeInfoCache.NodeCountDown()
					break
				}

				for k, v := range pv.ObjectMeta.Labels {
					if !volumeZoneLabels.Has(k) {
						continue
					}
					nodeV := nodeConstraints[k]
					volumeVSet, err := volumehelpers.LabelZonesToSet(v)
					if err != nil {
						r.KETI_LOG_L1(fmt.Sprintf("[error] Failed to parse label, ignoring the label - label %s:%s / err: %s", k, v, err))
						continue
					}

					if !volumeVSet.Has(nodeV) {
						// klog.V(10).InfoS("Won't schedule pod onto node due to volume (mismatch on label key)", "pod", klog.KObj(pod), "node", klog.KObj(node), "PV", klog.KRef("", pvName), "PVLabelKey", k)
						filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), ErrReasonConflict, nil}
						nodeinfo.PluginResult.FilterNode(nodeName, filterState)
						nodeInfoCache.NodeCountDown()
						break
					}
				}
			}
		}
	}

}

// New initializes a new plugin and returns it.
func NoVolumeZoneConflict_() NoVolumeZoneConflict {
	hostConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)
	informerFactory := informers.NewSharedInformerFactory(hostKubeClient, 0)

	pvLister := informerFactory.Core().V1().PersistentVolumes().Lister()
	pvcLister := informerFactory.Core().V1().PersistentVolumeClaims().Lister()
	scLister := informerFactory.Storage().V1().StorageClasses().Lister()
	return NoVolumeZoneConflict{
		pvLister,
		pvcLister,
		scLister,
	}
}
