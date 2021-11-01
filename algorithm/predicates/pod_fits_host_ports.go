package predicates

import (
	"fmt"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"
	"log"

	corev1 "k8s.io/api/core/v1"
)

func PodFitsHostPorts(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	fmt.Println("[step 1-3] Filtering > PodFitsHostPorts")

	wantPorts := getContainerPorts(newPod.Pod)
	if len(wantPorts) == 0 {
		return nil
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if !fitPorts(wantPorts, nodeinfo) {
				nodeinfo.FilterNode()
			}
		}
	}

	//no node to allocate
	if *resource.AvailableNodeCount == 0 {
		message := fmt.Sprintf("pod (%s) failed to fit in any node", newPod.Pod.ObjectMeta.Name)
		log.Println(message)
		event := postevent.MakeNoNodeEvent(newPod, message)
		err := postevent.PostEvent(event)
		if err != nil {
			fmt.Println("PodFitsHostPorts error: ", err)
			return err
		}
		return err
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
