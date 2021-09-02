package postevent

import (
	"context"
	"fmt"
	"gpu-scheduler/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func MakeNoNodeEvent(newPod *corev1.Pod, message string) *corev1.Event {
	event := &corev1.Event{
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
			Name:      newPod.Name,
			Namespace: "default",
			UID:       newPod.UID,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: newPod.Name + "-",
			Name:         newPod.Name,
		},
	}
	return event
}

func MakeBindEvent(pod *corev1.Pod, message string) *corev1.Event {
	event := &corev1.Event{
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
			Name:      pod.Name,
			Namespace: pod.Namespace,
			UID:       pod.UID,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pod.Name + "-",
			Name:         pod.Name,
		},
	}
	return event
}

func PostEvent(event *corev1.Event) error {
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)
	_, err := host_kubeClient.CoreV1().Events(event.InvolvedObject.Namespace).Update(context.TODO(), event, metav1.UpdateOptions{})
	if err != nil {
		fmt.Println("post event error: ", err)
		return err
	}
	return nil
}
