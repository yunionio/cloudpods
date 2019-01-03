package cloudprovider

type SLoadbalancerAccessControlListEntry struct {
	CIDR    string
	Comment string
}

type SLoadbalancerAccessControlList struct {
	Name   string
	Entrys []SLoadbalancerAccessControlListEntry
}
