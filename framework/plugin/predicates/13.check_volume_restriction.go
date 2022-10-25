package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type CheckVolumeRestriction struct{}

func (pl CheckVolumeRestriction) Name() string {
	return "CheckVolumeRestriction"
}

func (pl CheckVolumeRestriction) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#13. %s", pl.Name()))
}

func (pl CheckVolumeRestriction) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {

		}
	}
}
