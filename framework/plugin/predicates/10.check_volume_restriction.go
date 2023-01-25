package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"

	v1 "k8s.io/api/core/v1"
)

type CheckVolumeRestriction struct{}

func (pl CheckVolumeRestriction) Name() string {
	return "CheckVolumeRestriction"
}

func (pl CheckVolumeRestriction) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#10. %s", pl.Name()))
}

func (pl CheckVolumeRestriction) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			for i := range newPod.Pod.Spec.Volumes {
				v := &newPod.Pod.Spec.Volumes[i]
				// fast path if there is no conflict checking targets.
				if v.GCEPersistentDisk == nil && v.AWSElasticBlockStore == nil && v.RBD == nil && v.ISCSI == nil {
					continue
				}

				for _, ev := range nodeinfo.Pods {
					if isVolumeConflict(v, ev.Pod) {
						reason := "node(s) had no available disk"
						filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
						nodeinfo.PluginResult.FilterNode(nodeName, filterState)
						nodeInfoCache.NodeCountDown()
						break
					}
				}
			}
		}
	}
}

func isVolumeConflict(volume *v1.Volume, pod *v1.Pod) bool {
	for _, existingVolume := range pod.Spec.Volumes {
		// Same GCE disk mounted by multiple pods conflicts unless all pods mount it read-only.
		if volume.GCEPersistentDisk != nil && existingVolume.GCEPersistentDisk != nil {
			disk, existingDisk := volume.GCEPersistentDisk, existingVolume.GCEPersistentDisk
			if disk.PDName == existingDisk.PDName && !(disk.ReadOnly && existingDisk.ReadOnly) {
				return true
			}
		}

		if volume.AWSElasticBlockStore != nil && existingVolume.AWSElasticBlockStore != nil {
			if volume.AWSElasticBlockStore.VolumeID == existingVolume.AWSElasticBlockStore.VolumeID {
				return true
			}
		}

		if volume.ISCSI != nil && existingVolume.ISCSI != nil {
			iqn := volume.ISCSI.IQN
			eiqn := existingVolume.ISCSI.IQN
			// two ISCSI volumes are same, if they share the same iqn. As iscsi volumes are of type
			// RWO or ROX, we could permit only one RW mount. Same iscsi volume mounted by multiple Pods
			// conflict unless all other pods mount as read only.
			if iqn == eiqn && !(volume.ISCSI.ReadOnly && existingVolume.ISCSI.ReadOnly) {
				return true
			}
		}

		if volume.RBD != nil && existingVolume.RBD != nil {
			mon, pool, image := volume.RBD.CephMonitors, volume.RBD.RBDPool, volume.RBD.RBDImage
			emon, epool, eimage := existingVolume.RBD.CephMonitors, existingVolume.RBD.RBDPool, existingVolume.RBD.RBDImage
			// two RBDs images are the same if they share the same Ceph monitor, are in the same RADOS Pool, and have the same image name
			// only one read-write mount is permitted for the same RBD image.
			// same RBD image mounted by multiple Pods conflicts unless all Pods mount the image read-only
			if haveOverlap(mon, emon) && pool == epool && image == eimage && !(volume.RBD.ReadOnly && existingVolume.RBD.ReadOnly) {
				return true
			}
		}
	}

	return false
}
func haveOverlap(a1, a2 []string) bool {
	if len(a1) > len(a2) {
		a1, a2 = a2, a1
	}
	m := map[string]bool{}

	for _, val := range a1 {
		m[val] = true
	}
	for _, val := range a2 {
		if _, ok := m[val]; ok {
			return true
		}
	}

	return false
}
