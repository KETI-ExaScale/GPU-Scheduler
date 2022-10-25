package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type PodFitsHost struct{}

func (pl PodFitsHost) Name() string {
	return "PodFitsHost"
}

func (pl PodFitsHost) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#1. %s", pl.Name()))
}

func (pl PodFitsHost) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	if len(newPod.Pod.Spec.NodeName) == 0 {
		return
	}

	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if newPod.Pod.Spec.NodeName != nodeName {
				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
				nodeInfoCache.NodeCountDown()
				newPod.FilterNode(pl.Name())
			}
		}
	}
}
