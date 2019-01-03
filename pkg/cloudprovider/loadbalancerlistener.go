package cloudprovider

type SLoadbalancerListener struct {
	Name                    string
	LoadbalancerID          string
	ListenerType            string
	ListenerPort            int
	BackendGroupID          string
	Scheduler               string
	AccessControlListStatus string
	AccessControlListType   string
	AccessControlListID     string
	EnableHTTP2             bool
	CertificateID           string
}
