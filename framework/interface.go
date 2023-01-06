package framework

import (
	"gpu-scheduler/framework/plugin/predicates"
	"gpu-scheduler/framework/plugin/priorities"
	r "gpu-scheduler/resourceinfo"
)

type GPUSchedulerInterface interface {
	//RunPreFilteringPlugins()
	RunFilteringPlugins(*r.NodeCache, *r.QueuedPodInfo)
	//RunPostFilteringPlugins()
	RunScoringPlugins(*r.NodeCache, *r.QueuedPodInfo)
}

func GPUPodSpreadFramework() GPUSchedulerInterface {
	return &GPUSchedulerFramework{
		Filtering: []FilterPlugin{
			predicates.PodFitsHost{},
			predicates.CheckNodeUnschedulable{},
			predicates.PodFitsHostPorts{},
			predicates.NodeFitsGPUMemory{},
			predicates.NodeFitsGPUCount{},
			predicates.PodFitsNodeResources{},
			predicates.MatchNodeSelector{},
			// predicates.PodToleratesNodeTaints{},
			predicates.PodTopologySpread{},
			predicates.InterPodAffinity{},
			predicates.CheckVolumeBinding{},
			predicates.NoVolumeZoneConflict{},
			predicates.CheckVolumeRestriction{},
			predicates.CheckNodeReserved{},
		},
		Scoring: []ScorePlugin{
			priorities.NodeAffinity{},
			priorities.TaintToleration{},
			priorities.SelectorSpread{},
			priorities.InterPodAffinity{},
			priorities.PodTopologySpread{},
			priorities.ImageLocality{},
			priorities.NodeResourcesFit{},
			priorities.BalancedNodeResourceAllocation{},
			priorities.VolumeBinding{},
			priorities.NodeMetricAnalysis{},
			priorities.SetGPUFlopsScore{},
			priorities.AllocatedPodCountInGPU{},
			priorities.GPUUtilization{},
			priorities.GPUMemoryUsage{},
			priorities.GPUMerticAnalysis{},
			priorities.GPUTemperature{},
			priorities.GPUPower{},
			priorities.GPUBandwidth{},
			// priorities.GPUDirectStoragePriority{},
			// priorities.BalancedGPUProcessType{},
		},
	}
}

func GPUPodBinpackFramework() GPUSchedulerInterface {
	return &GPUSchedulerFramework{
		Filtering: []FilterPlugin{
			predicates.PodFitsHost{},
			predicates.CheckNodeUnschedulable{},
			predicates.PodFitsHostPorts{},
			predicates.NodeFitsGPUMemory{},
			predicates.NodeFitsGPUCount{},
			predicates.PodFitsNodeResources{},
			predicates.MatchNodeSelector{},
			// predicates.PodToleratesNodeTaints{},
			predicates.PodTopologySpread{},
			predicates.InterPodAffinity{},
			predicates.CheckVolumeBinding{},
			predicates.NoVolumeZoneConflict{},
			predicates.CheckVolumeRestriction{},
			predicates.CheckNodeReserved{},
		},
		Scoring: []ScorePlugin{
			priorities.NodeAffinity{},
			priorities.TaintToleration{},
			priorities.SelectorSpread{},
			priorities.InterPodAffinity{},
			priorities.PodTopologySpread{},
			priorities.ImageLocality{},
			priorities.NodeResourcesFit{},
			priorities.BalancedNodeResourceAllocation{},
			priorities.VolumeBinding{},
			priorities.NodeMetricAnalysis{},
			priorities.SetGPUFlopsScore{},
			priorities.AllocatedPodCountInGPUBinpack{},
			// priorities.GPUUtilization{},
			priorities.GPUMemoryUsageBinpack{},
			// priorities.GPUMerticAnalysis{},
			// priorities.GPUTemperature{},
			// priorities.GPUPower{},
			// priorities.GPUBandwidth{},
			// priorities.GPUDirectStoragePriority{},
			// priorities.BalancedGPUProcessType{},
		},
	}
}

func NonGPUPodFramework() GPUSchedulerInterface {
	return &GPUSchedulerFramework{
		Filtering: []FilterPlugin{
			predicates.PodFitsHost{},
			predicates.CheckNodeUnschedulable{},
			predicates.PodFitsHostPorts{},
			predicates.PodFitsNodeResources{},
			predicates.MatchNodeSelector{},
			// predicates.PodToleratesNodeTaints{},
			predicates.PodTopologySpread{},
			predicates.InterPodAffinity{},
			predicates.CheckVolumeBinding{},
			predicates.NoVolumeZoneConflict{},
			predicates.CheckVolumeRestriction{},
			predicates.CheckNodeReserved{},
		},
		Scoring: []ScorePlugin{
			priorities.NodeAffinity{},
			priorities.TaintToleration{},
			priorities.SelectorSpread{},
			priorities.InterPodAffinity{},
			priorities.PodTopologySpread{},
			priorities.ImageLocality{},
			priorities.NodeResourcesFit{},
			priorities.BalancedNodeResourceAllocation{},
			priorities.VolumeBinding{},
			priorities.NodeMetricAnalysis{},
		},
	}
}

func InitNodeScoreFramework() GPUSchedulerInterface {
	return &GPUSchedulerFramework{
		Filtering: []FilterPlugin{}, //only scoring
		Scoring: []ScorePlugin{
			priorities.NodeResourcesFit{},
			priorities.BalancedNodeResourceAllocation{},
			priorities.NodeMetricAnalysis{},
			priorities.SetGPUFlopsScore{},
			priorities.AllocatedPodCountInGPU{},
			priorities.GPUUtilization{},
			priorities.GPUMemoryUsage{},
			priorities.GPUMerticAnalysis{},
			priorities.GPUTemperature{},
			priorities.GPUPower{},
			priorities.GPUBandwidth{},
		},
	}
}

type GPUSchedulerFramework struct {
	Filtering []FilterPlugin
	Scoring   []ScorePlugin
}

type PluginName interface {
	Name() string
}

type FilterPlugin interface {
	PluginName
	Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo)
	Debugg()
}

type ScorePlugin interface {
	PluginName
	Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo)
	Debugg(nodeInfoCache *r.NodeCache)
}

func (sf GPUSchedulerFramework) RunFilteringPlugins(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, fp := range sf.Filtering {
		fp.Debugg()
		fp.Filter(nodeInfoCache, newPod)
		if nodeInfoCache.AvailableNodeCount == 0 {
			return
		}
	}
}

func (sf GPUSchedulerFramework) RunScoringPlugins(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, sp := range sf.Scoring {
		sp.Score(nodeInfoCache, newPod)
		sp.Debugg(nodeInfoCache)
	}
}
