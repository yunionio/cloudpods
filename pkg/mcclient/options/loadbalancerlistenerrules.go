package options

type LoadbalancerListenerRuleCreateOptions struct {
	NAME         string
	Listener     string `required:true`
	BackendGroup string
	Domain       string
	Path         string
}

type LoadbalancerListenerRuleListOptions struct {
	BaseListOptions
	Listener string
	Domain   string
	Path     string
}

type LoadbalancerListenerRuleUpdateOptions struct {
	ID           string
	BackendGroup string
}

type LoadbalancerListenerRuleGetOptions struct {
	ID string
}

type LoadbalancerListenerRuleDeleteOptions struct {
	ID string
}

type LoadbalancerListenerRuleActionStatusOptions struct {
	ID     string
	Status string `choices:"enabled|disabled"`
}
