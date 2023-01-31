package priorities

import (
	r "gpu-scheduler/resourceinfo"
)

type UnsatisfiableConstraintAction string

func DefaultNormalizeScore(maxPriority int64, reverse bool, nodeinfolist []*r.NodeInfo) bool {
	var maxCount int64
	for _, nodeinfo := range nodeinfolist {
		if !nodeinfo.PluginResult.IsFiltered {
			if int64(nodeinfo.PluginResult.NodeScore) > maxCount {
				maxCount = int64(nodeinfo.PluginResult.NodeScore)
			}
		}
	}

	if maxCount == 0 {
		if reverse {
			for _, nodeinfo := range nodeinfolist {
				if !nodeinfo.PluginResult.IsFiltered {
					nodeinfo.PluginResult.NodeScore = int(maxPriority)
				}
			}
		}
		return true
	}

	for _, nodeinfo := range nodeinfolist {
		if !nodeinfo.PluginResult.IsFiltered {
			score := int64(nodeinfo.PluginResult.NodeScore)
			score = maxPriority * score / maxCount
			if reverse {
				score = maxPriority - score
			}
			nodeinfo.PluginResult.NodeScore = int(score)

		}
	}

	return true
}
