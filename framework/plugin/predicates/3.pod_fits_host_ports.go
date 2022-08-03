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
	fmt.Println("#3. ", pl.Name())
}

func (pl PodFitsHostPorts) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {

	wantPorts := getContainerPorts(newPod.Pod)
	if len(wantPorts) == 0 {
		return
	}

	fmt.Print("- nodes: {")
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if !fitsPorts(wantPorts, nodeinfo) {
				nodeinfo.PluginResult.FilterNode(pl.Name())
				nodeInfoCache.NodeCountDown()
			}
		}
		if !nodeinfo.PluginResult.IsFiltered {
			fmt.Print(nodeName, ", ")
		}
	}
	fmt.Println("}")
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
