package framework

import (
	"gpu-scheduler/framework/plugin/predicates"
	r "gpu-scheduler/resourceinfo"
)

type GPUSchedulerInterface interface {
	RunNodeFilteringPlugins(*r.NodeCache, *r.QueuedPodInfo)
	RunGPUTemperatureFilteringPlugin(*r.NodeCache, *r.QueuedPodInfo)
	RunGPUCountFilteringPlugin(*r.NodeCache, *r.QueuedPodInfo)
}

func GPUPodFramework() GPUSchedulerInterface {
	return &GPUSchedulerFramework{
		NodeFiltering: []FilterPlugin{
			predicates.PodFitsHost{},
			predicates.CheckNodeUnschedulable{},
			predicates.PodFitsHostPorts{},
			predicates.PodToleratesNodeTaints{},
			predicates.PodTopologySpread_(),
			predicates.InterPodAffinity_(),
			predicates.MatchNodeSelector{},
			predicates.CheckVolumeBinding{},
			predicates.NoVolumeZoneConflict_(),
			predicates.CheckVolumeRestriction{},
			predicates.PodFitsNodeResources{},
			predicates.CheckNodeReserved{},
		},
		GPUTemperatureFiltering: []FilterPlugin{
			predicates.CheckGPUTemperature{},
		},
		GPUCountFiltering: []FilterPlugin{
			predicates.NodeFitsGPUCount{},
		},
	}
}

type GPUSchedulerFramework struct {
	NodeFiltering           []FilterPlugin
	GPUTemperatureFiltering []FilterPlugin
	GPUCountFiltering       []FilterPlugin
}

type PluginName interface {
	Name() string
}

type FilterPlugin interface {
	PluginName
	Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo)
	Debugg()
}

func (sf GPUSchedulerFramework) RunNodeFilteringPlugins(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nfp := range sf.NodeFiltering {
		nfp.Debugg()
		nfp.Filter(nodeInfoCache, newPod)
		if nodeInfoCache.AvailableNodeCount == 0 {
			return
		}
	}
}

func (sf GPUSchedulerFramework) RunGPUTemperatureFilteringPlugin(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	sf.GPUTemperatureFiltering[0].Debugg()
	sf.GPUTemperatureFiltering[0].Filter(nodeInfoCache, newPod)
	if nodeInfoCache.AvailableNodeCount == 0 {
		return
	}
}

func (sf GPUSchedulerFramework) RunGPUCountFilteringPlugin(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	sf.GPUCountFiltering[0].Debugg()
	sf.GPUTemperatureFiltering[0].Filter(nodeInfoCache, newPod)
	if nodeInfoCache.AvailableNodeCount == 0 {
		return
	}
}
