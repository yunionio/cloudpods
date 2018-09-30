package modules

type LoadbalancerBackendGroupManager struct {
	ResourceManager
}

var (
	LoadbalancerBackendGroups LoadbalancerBackendGroupManager
)

func init() {
	LoadbalancerBackendGroups = LoadbalancerBackendGroupManager{
		NewComputeManager(
			"loadbalancerbackendgroup",
			"loadbalancerbackendgroups",
			[]string{
				"id",
				"name",
				"loadbalancer_id",
			},
			[]string{"tenant"},
		),
	}
	registerCompute(&LoadbalancerBackendGroups)
}
