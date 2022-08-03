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
	fmt.Println("#12. ", pl.Name())
}

func (pl NoVolumeZoneConflict) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	fmt.Print("- nodes: {")
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {

		}
		if !nodeinfo.PluginResult.IsFiltered {
			fmt.Print(nodeName, ", ")
		}
	}
	fmt.Println("}")

}
