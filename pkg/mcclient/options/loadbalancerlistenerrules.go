package options

type LoadbalancerListenerRuleCreateOptions struct {
	NAME         string
	Listener     string `required:"true"`
	BackendGroup string
	Domain       string
	Path         string
}

type LoadbalancerListenerRuleListOptions struct {
	BaseListOptions

	BackendGroup string
	Listener     string
	Domain       string
	Path         string
}

type LoadbalancerListenerRuleUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	BackendGroup string
}

type LoadbalancerListenerRuleGetOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListenerRuleDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListenerRuleActionStatusOptions struct {
	ID     string `json:"-"`
	Status string `choices:"enabled|disabled"`
}
