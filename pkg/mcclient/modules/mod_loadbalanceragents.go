package modules

type LoadbalancerAgentManager struct {
	ResourceManager
}

var (
	LoadbalancerAgents LoadbalancerAgentManager
)

func init() {
	LoadbalancerAgents = LoadbalancerAgentManager{
		NewComputeManager(
			"loadbalanceragent",
			"loadbalanceragents",
			[]string{
				"id",
				"name",

				"hb_last_seen",
				"hb_timeout",

				"loadbalancers",
				"loadbalancer_listeners",
				"loadbalancer_listener_rules",
				"loadbalancer_backend_groups",
				"loadbalancer_backends",
				"loadbalancer_acls",
				"loadbalancer_certificates",
			},
			[]string{},
		),
	}
	registerCompute(&LoadbalancerAgents)
}
