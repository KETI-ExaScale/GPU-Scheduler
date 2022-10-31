package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type PodToleratesNodeTaints struct{}

func (pl PodToleratesNodeTaints) Name() string {
	return "PodToleratesNodeTaints"
}

func (pl PodToleratesNodeTaints) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#8. %s", pl.Name()))
}

func (pl PodToleratesNodeTaints) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			fmt.Println("nodename: ", nodeName, nodeinfo.Node().Name)
			for _, taint := range nodeinfo.Node().Spec.Taints {
				fmt.Println("node taints: ", taint)
				tolerated := false
				for _, toleration := range newPod.Pod.Spec.Tolerations {
					fmt.Println("pod toleration: ", toleration)
					if toleration.ToleratesTaint(&taint) {
						tolerated = true
						break
					}
				}
				if !tolerated {
					nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
					nodeInfoCache.NodeCountDown()
					newPod.FilterNode(pl.Name())
					break
				}
			}
		}
	}
}
