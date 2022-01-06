package resourceinfo

import (
	"context"
	"fmt"
	"gpu-scheduler/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeNoNodeEvent(newPod *Pod, message string) *corev1.Event {
	return &corev1.Event{
		Count:          1,
		Message:        message,
		Reason:         "FailedScheduling",
		LastTimestamp:  metav1.Now(),
		FirstTimestamp: metav1.Now(),
		Type:           "Warning",
		Source: corev1.EventSource{
			Component: config.SchedulerName,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Pod",
			Name:      newPod.Pod.Name,
			Namespace: "default",
			UID:       newPod.Pod.UID,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: newPod.Pod.Name + "-",
			Name:         newPod.Pod.Name,
		},
	}
}

func MakeBindEvent(pod *Pod, message string) *corev1.Event {
	return &corev1.Event{
		Count:          1,
		Message:        message,
		Reason:         "Scheduled",
		LastTimestamp:  metav1.Now(),
		FirstTimestamp: metav1.Now(),
		Type:           "Normal",
		Source: corev1.EventSource{
			Component: config.SchedulerName,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Pod",
			Name:      pod.Pod.Name,
			Namespace: pod.Pod.Namespace,
			UID:       pod.Pod.UID,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pod.Pod.Name + "-",
			Name:         pod.Pod.Name,
		},
	}
}

func PostEvent(event *corev1.Event) {
	_, err := config.Host_kubeClient.CoreV1().Events(event.InvolvedObject.Namespace).Update(context.TODO(), event, metav1.UpdateOptions{})
	if err != nil {
		fmt.Println("post event failed")
	}
}
