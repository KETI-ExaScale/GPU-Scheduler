package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type NoVolumeZoneConflict struct{}

func (pl NoVolumeZoneConflict) Name() string {
	return "NoVolumeZoneConflict"
}

func (pl NoVolumeZoneConflict) Debugg() {
	fmt.Println("F#12.", pl.Name())
}

func (pl NoVolumeZoneConflict) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {

		}
	}

}
