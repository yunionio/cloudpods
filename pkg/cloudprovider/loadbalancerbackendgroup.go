package cloudprovider

type SLoadbalancerBackendGroup struct {
	Name      string
	GroupType string
	Backends  []SLoadbalancerBackend
}
