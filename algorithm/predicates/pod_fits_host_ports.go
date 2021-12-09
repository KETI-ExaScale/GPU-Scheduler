package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
)

func PodFitsHostPorts(newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-3] Filtering > PodFitsHostPorts")
	}

	wantPorts := getContainerPorts(newPod.Pod)
	if len(wantPorts) == 0 {
		return nil
	}

	for _, nodeinfo := range resource.NodeInfoList {
		if !nodeinfo.IsFiltered {
			if !fitPorts(wantPorts, nodeinfo) {
				nodeinfo.FilterNode()
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> pod_fits_host_ports")
	}

	return nil
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

func fitPorts(wantPorts []*corev1.ContainerPort, nodeinfo *resource.NodeInfo) bool {
	for _, wantPort := range wantPorts {
		if wantPort.HostPort <= 0 {
			continue
		}
		for i := range nodeinfo.Pods {
			pod := nodeinfo.Pods[i]

			for j := range pod.Spec.Containers {
				container := &pod.Spec.Containers[j]

				for k := range container.Ports {
					existingPort := &container.Ports[k]

					if existingPort.HostPort == wantPort.HostPort {
						return false
					}
				}
			}
		}
	}

	return true
}
