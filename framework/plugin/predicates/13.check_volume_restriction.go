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
	fmt.Println("#13. ", pl.Name())
}

func (pl CheckVolumeRestriction) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
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
