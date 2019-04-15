package modules

type LoadbalancerManager struct {
	ResourceManager
}

var (
	Loadbalancers LoadbalancerManager
)

func init() {
	Loadbalancers = LoadbalancerManager{
		NewComputeManager(
			"loadbalancer",
			"loadbalancers",
			[]string{
				"id",
				"name",
				"status",
				"address_type",
				"address",
				"network_type",
				"network_id",
				"vpc_id",
				"zone_id",
				"charge_type",
				"bandwidth",
				"backend_group_id",
				"loadbalancer_spec",
				"cloudregion_id",
			},
			[]string{"tenant"},
		),
	}
	registerCompute(&Loadbalancers)

}
