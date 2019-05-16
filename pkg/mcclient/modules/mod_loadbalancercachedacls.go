package modules

type LoadbalancerCachedAclManager struct {
	ResourceManager
}

var (
	LoadbalancerCachedAcls LoadbalancerCachedAclManager
)

func init() {
	LoadbalancerCachedAcls = LoadbalancerCachedAclManager{
		NewComputeManager(
			"cachedloadbalanceracl",
			"cachedloadbalanceracls",
			[]string{
				"id",
				"acl_id",
				"name",
				"acl_entries",
			},
			[]string{"tenant"},
		),
	}
	registerCompute(&LoadbalancerCachedAcls)
}
