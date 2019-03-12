package options

type LoadbalancerCreateOptions struct {
	NAME             string
	Network          string
	Address          string
	AddressType      string `choices:"intranet|internet"`
	LoadbalancerSpec string `choices:"slb.s1.small|slb.s2.small|slb.s2.medium|slb.s3.small|slb.s3.medium|slb.s3.large"`
	ChargeType       string `choices:"traffic|bandwidth"`
	Zone             string
	ManagerId        string
}

type LoadbalancerGetOptions struct {
	ID string `json:-`
}

type LoadbalancerUpdateOptions struct {
	ID   string `json:-`
	Name string

	BackendGroup string
}

type LoadbalancerDeleteOptions struct {
	ID string `json:-`
}

type LoadbalancerPurgeOptions struct {
	ID string `json:-`
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
	ID     string `json:-`
	Status string `choices:"enabled|disabled"`
}

type LoadbalancerActionSyncStatusOptions struct {
	ID string `json:-`
}
