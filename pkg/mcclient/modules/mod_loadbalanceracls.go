package modules

type LoadbalancerAclManager struct {
	ResourceManager
}

var (
	LoadbalancerAcls LoadbalancerAclManager
)

func init() {
	LoadbalancerAcls = LoadbalancerAclManager{
		NewComputeManager(
			"loadbalanceracl",
			"loadbalanceracls",
			[]string{
				"id",
				"name",
				"acl_entries",
			},
			[]string{"tenant"},
		),
	}
	registerCompute(&LoadbalancerAcls)
}
