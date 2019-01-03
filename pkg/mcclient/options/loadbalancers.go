package options

type LoadbalancerCreateOptions struct {
	NAME             string
	Network          string
	Address          string
	LoadbalancerSpec string `choices:"slb.s1.small|slb.s2.small|slb.s2.medium|slb.s3.small|slb.s3.medium|slb.s3.large"`
	ChargeType       string `choices:"traffic|bandwidth"`
}

type LoadbalancerGetOptions struct {
	ID string
}

type LoadbalancerUpdateOptions struct {
	ID   string
	Name string

	BackendGroup string
}

type LoadbalancerDeleteOptions struct {
	ID string
}

type LoadbalancerPurgeOptions struct {
	ID string
}

type LoadbalancerListOptions struct {
	BaseListOptions
	Zone         string
	Address      string
	AddressType  string `choices:"intranet|internet"`
	NetworkType  string `choices:"classic|vpc"`
	Network      string
	BackendGroup string
}

type LoadbalancerActionStatusOptions struct {
	ID     string
	Status string `choices:"enabled|disabled"`
}
