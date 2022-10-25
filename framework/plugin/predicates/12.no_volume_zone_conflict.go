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
	r.KETI_LOG_L2(fmt.Sprintf("F#12. %s", pl.Name()))
}

func (pl NoVolumeZoneConflict) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {

		}
	}

}
