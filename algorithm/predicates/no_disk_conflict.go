package predicates

import (
	"fmt"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"
	"log"

	corev1 "k8s.io/api/core/v1"
)

func NoDiskConflict(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	fmt.Println("[step 1-7] Filtering > NoDiskConflict")

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			conflict := false
			for _, volume := range newPod.Pod.Spec.Volumes {
				for _, ev := range nodeinfo.Pods {
					if isVolumeConflict(volume, ev) {
						conflict = true
						break
					}
				}
				if conflict {
					nodeinfo.FilterNode()
					break
				}
			}
		}
	}

	//no node to allocate
	if *resource.AvailableNodeCount == 0 {
		message := fmt.Sprintf("pod (%s) failed to fit in any node", newPod.Pod.ObjectMeta.Name)
		log.Println(message)
		event := postevent.MakeNoNodeEvent(newPod, message)
		err := postevent.PostEvent(event)
		if err != nil {
			fmt.Println("NoDiskConflict error: ", err)
			return err
		}
		return err
	}

	return nil
}

func isVolumeConflict(volume corev1.Volume, pod *corev1.Pod) bool {
	// fast path if there is no conflict checking targets.
	if volume.GCEPersistentDisk == nil && volume.AWSElasticBlockStore == nil && volume.RBD == nil && volume.ISCSI == nil {
		return false
	}

	for _, existingVolume := range pod.Spec.Volumes {
		// case 1) GCEPersistentDisk
		if volume.GCEPersistentDisk != nil && existingVolume.GCEPersistentDisk != nil {
			disk, existingDisk := volume.GCEPersistentDisk, existingVolume.GCEPersistentDisk
			if disk.PDName == existingDisk.PDName && !(disk.ReadOnly && existingDisk.ReadOnly) {
				return true
			}
		}

		// case 2) AWSElasticBlockStore
		if volume.AWSElasticBlockStore != nil && existingVolume.AWSElasticBlockStore != nil {
			if volume.AWSElasticBlockStore.VolumeID == existingVolume.AWSElasticBlockStore.VolumeID {
				return true
			}
		}

		// case 3) ISCSI
		if volume.ISCSI != nil && existingVolume.ISCSI != nil {
			iqn := volume.ISCSI.IQN
			eiqn := existingVolume.ISCSI.IQN
			if iqn == eiqn && !(volume.ISCSI.ReadOnly && existingVolume.ISCSI.ReadOnly) {
				return true
			}
		}

		// case 4) RBD
		if volume.RBD != nil && existingVolume.RBD != nil {
			mon, pool, image := volume.RBD.CephMonitors, volume.RBD.RBDPool, volume.RBD.RBDImage
			emon, epool, eimage := existingVolume.RBD.CephMonitors, existingVolume.RBD.RBDPool, existingVolume.RBD.RBDImage
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
