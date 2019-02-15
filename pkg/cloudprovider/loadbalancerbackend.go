package cloudprovider

type SLoadbalancerBackend struct {
	Index       int
	Weight      int
	Port        int
	ID          string
	Name        string
	ExternalID  string
	BackendType string
	BackendRole string
	Address     string
	ZoneId      string
	HostName    string
}
