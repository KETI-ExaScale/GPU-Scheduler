package config

var (
	ApiHost             = "10.0.5.21:6443"
	BindingsEndpoint    = "/api/v1/namespaces/%s/pods/%s/binding/"
	AnnotationsEndpoint = "/api/v1/namespaces/%s/pods/%s"
	EventsEndpoint      = "/api/v1/namespaces/%s/events"
	NodesEndpoint       = "/api/v1/nodes"
	PodsEndpoint        = "/api/v1/pods"
	WatchPodsEndpoint   = "/api/v1/watch/pods"
)

const SchedulerName = "gpu-scheduler"
