package v1alpha1

// import (
// 	corev1 "k8s.io/api/core/v1"
// )

// const (
// 	filter  = "Filter"
// 	score   = "Score"
// 	bind    = "Bind"
// 	reserve = "Reserve"
// )

// type Framework struct {
// 	filterPlugins []FilterPlugin
// 	scorePlugins  []ScorePlugin
// 	bindPlugins   []BindPlugin
// }

// var _ gpuschedulerFramework = &Framework{}

// // RunFilterPlugins runs the set of configured Filter plugins for pod on
// // the given node. If any of these plugins doesn't return "Success", the
// // given node is not suitable for running pod.
// // Meanwhile, the failure message and status are set for the given node.
// func (f *Framework) RunFilterPlugins(pod *corev1.Pod)
