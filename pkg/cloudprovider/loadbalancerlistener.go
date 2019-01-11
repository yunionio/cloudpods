package cloudprovider

type SLoadbalancerListener struct {
	Name                    string
	LoadbalancerID          string
	ListenerType            string
	ListenerPort            int
	BackendGroupType        string
	BackendGroupID          string
	Scheduler               string
	AccessControlListStatus string
	AccessControlListType   string
	AccessControlListID     string
	EnableHTTP2             bool
	CertificateID           string
	Bandwidth               int
	Description             string
	EstablishedTimeout      int

	HealthCheck         string
	HealthCheckTimeout  int
	HealthCheckDomain   string
	HealthCheckHttpCode string
	HealthCheckURI      string
	HealthCheckInterval int

	HealthCheckRise int
	HealthCheckFail int

	StickySession              string
	StickySessionCookie        string
	StickySessionType          string
	StickySessionCookieTimeout int

	ForwardPort   int
	XForwardedFor bool
	Gzip          bool

	TLSCipherPolicy string
}

type SLoadbalancerListenerRule struct {
	Name             string
	Domain           string
	Path             string
	BackendGroupID   string
	BackendGroupType string
}
