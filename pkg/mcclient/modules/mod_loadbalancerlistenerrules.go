package modules

type LoadbalancerListenerRuleManager struct {
	ResourceManager
}

var (
	LoadbalancerListenerRules LoadbalancerListenerRuleManager
)

func init() {
	LoadbalancerListenerRules = LoadbalancerListenerRuleManager{
		NewComputeManager(
			"loadbalancerlistenerrule",
			"loadbalancerlistenerrules",
			[]string{
				"id",
				"name",
				"listener_id",
				"status",
				"domain",
				"path",
				"backend_id",
			},
			[]string{"tenant"},
		),
	}
	registerCompute(&LoadbalancerListenerRules)
}
