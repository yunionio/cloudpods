package modules

type LoadbalancerBackendManager struct {
	ResourceManager
}

var (
	LoadbalancerBackends LoadbalancerBackendManager
)

func init() {
	LoadbalancerBackends = LoadbalancerBackendManager{
		NewComputeManager(
			"loadbalancerbackend",
			"loadbalancerbackends",
			[]string{
				"id",
				"name",
				"backend_group_id",
				"backend_id",
				"backend_type",
				"address",
				"port",
				"weight",
			},
			[]string{"tenant"},
		),
	}
	registerCompute(&LoadbalancerBackends)
}
