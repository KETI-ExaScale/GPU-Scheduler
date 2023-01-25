package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
)

type PodFitsHostPorts struct{}

func (pl PodFitsHostPorts) Name() string {
	return "PodFitsHostPorts"
}

func (pl PodFitsHostPorts) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#3. %s", pl.Name()))
}

func (pl PodFitsHostPorts) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {

	wantPorts := getContainerPorts(newPod.Pod)
	if len(wantPorts) == 0 {
		return
	}

	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if !fitsPorts(wantPorts, nodeinfo) {
				reason := fmt.Sprintf("cannot use pod requested port : %v")
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
			}
		}
	}
}

// getContainerPorts returns the used host ports of Pods: if 'port' was used, a 'port:true' pair
func getContainerPorts(pods ...*corev1.Pod) []*corev1.ContainerPort {
	var ports []*corev1.ContainerPort
	for _, pod := range pods {
		for i := range pod.Spec.Containers {
			container := &pod.Spec.Containers[i]
			for j := range container.Ports {
				ports = append(ports, &container.Ports[j])
			}
		}
	}
	return ports
}

func fitsPorts(wantPorts []*corev1.ContainerPort, nodeinfo *r.NodeInfo) bool {
	// try to see whether existingPorts and wantPorts will conflict or not
	existingPorts := nodeinfo.UsedPorts
	for _, cp := range wantPorts {
		if existingPorts.CheckConflict(cp.HostIP, string(cp.Protocol), cp.HostPort) {
			return false
		}
	}
	return true
}
