package options

type LoadbalancerBackendCreateOptions struct {
	BackendGroup string `required:"true"`
	Backend      string `required:"true"`
	BackendType  string `default:"guest"`
	Port         *int   `required:"true"`
	Weight       *int   `default:"1"`
}

type LoadbalancerBackendListOptions struct {
	BaseListOptions
	BackendGroup string
	Backend      string
	BackendType  string
	Weight       *int
	Address      string
	Port         *int
}

type LoadbalancerBackendUpdateOptions struct {
	ID string

	Weight *int
	Port   *int
}

type LoadbalancerBackendGetOptions struct {
	ID string
}

type LoadbalancerBackendDeleteOptions struct {
	ID string
}
