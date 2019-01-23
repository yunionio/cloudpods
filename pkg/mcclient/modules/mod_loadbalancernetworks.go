package modules

var (
	Loadbalancernetworks JointResourceManager
)

func init() {
	Loadbalancernetworks = NewJointComputeManager(
		"loadbalancernetwork",
		"loadbalancernetworks",
		[]string{"Loadbalancer_ID", "Loadbalancer",
			"Network_ID", "Network", "Ip_Addr"},
		[]string{},
		&Loadbalancers,
		&Networks)
	registerCompute(&Loadbalancernetworks)
}
