package modules

type LoadbalancerListenerManager struct {
	ResourceManager
}

var (
	LoadbalancerListeners LoadbalancerListenerManager
)

func init() {
	LoadbalancerListeners = LoadbalancerListenerManager{
		NewComputeManager(
			"loadbalancerlistener",
			"loadbalancerlisteners",
			[]string{
				"id",
				"name",
				"loadbalancer_id",
				"status",
				"listener_type",
				"listener_port",
				"backend_port",
				"acl_status",
				"acl_type",
			},
			[]string{"tenant"},
		),
	}
	registerCompute(&LoadbalancerListeners)
}
