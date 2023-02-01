package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type CheckNodeReserved struct{}

func (pl CheckNodeReserved) Name() string {
	return "CheckNodeReserved"
}

func (pl CheckNodeReserved) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#14. %s", pl.Name()))
}

func (pl CheckNodeReserved) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if nodeinfo.Reserved {
				//filter
			}
		}
	}
}
