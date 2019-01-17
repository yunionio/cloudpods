package options

type LoadbalancerListenerCreateOptions struct {
	NAME string

	Loadbalancer      string `required:"true"`
	ListenerType      string `required:"true" choices:"tcp|udp|http|https"`
	ListenerPort      *int   `required:"true"`
	BackendServerPort *int
	BackendGroup      string

	Scheduler string `required:"true" choices:"rr|wrr|wlc|sch|tch"`

	ClientRequestTimeout  *int
	ClientIdleTimeout     *int
	BackendConnectTimeout *int
	BackendIdleTimeout    *int

	AclStatus string `choices:"on|off"`
	AclType   string `choices:"black|white"`
	Acl       string

	HealthCheck     string `choices:"on|off"`
	HealthCheckType string `choices:"tcp|http"`

	HealthCheckDomain   string
	HealthCheckURI      string
	HealthCheckHttpCode string

	HealthCheckRise     *int
	HealthCheckFall     *int
	HealthCheckInterval *int
	HealthCheckTimeout  *int

	HealthCheckReq string
	HealthCheckExp string

	StickySession              string
	StickySessionType          string
	StickySessionCookie        string
	StickySessionCookieTimeout *int

	XForwardedFor string `choices:"true|false"`
	Gzip          string `choices:"true|false"`

	Certificate     string
	TLSCipherPolicy string
	EnableHttp2     string `choices:"true|false"`

	HTTPRequestRate       *int
	HTTPRequestRatePerSrc *int
}

type LoadbalancerListenerListOptions struct {
	BaseListOptions

	Loadbalancer string
	ListenerType string `choices:"tcp|udp|http|https"`
	ListenerPort *int
	BackendGroup string

	Scheduler string `choices:"rr|wrr|wlc|sch|tch"`

	ClientRequestTimeout  *int
	ClientIdleTimeout     *int
	BackendConnectTimeout *int
	BackendIdleTimeout    *int

	AclStatus string `choices:"on|off"`
	AclType   string `choices:"black|white"`
	Acl       string

	HealthCheck     string `choices:"on|off"`
	HealthCheckType string `choices:"tcp|http"`

	HealthCheckDomain   string
	HealthCheckURI      string
	HealthCheckHttpCode string

	HealthCheckRise     *int
	HealthCheckFall     *int
	HealthCheckInterval *int
	HealthCheckTimeout  *int

	HealthCheckReq string
	HealthCheckExp string

	StickySession              string
	StickySessionType          string
	StickySessionCookie        string
	StickySessionCookieTimeout *int

	XForwardedFor string `choices:"true|false"`
	Gzip          string `choices:"true|false"`

	Certificate     string
	TLSCipherPolicy string
	EnableHttp2     string `choices:"true|false"`

	HTTPRequestRate       *int
	HTTPRequestRatePerSrc *int
}

type LoadbalancerListenerUpdateOptions struct {
	ID   string
	Name string

	BackendGroup string

	Scheduler string `choices:"rr|wrr|wlc|sch|tch"`

	ClientRequestTimeout  *int
	ClientIdleTimeout     *int
	BackendConnectTimeout *int
	BackendIdleTimeout    *int

	AclStatus string `choices:"on|off"`
	AclType   string `choices:"black|white"`
	Acl       string

	HealthCheck     string `choices:"on|off"`
	HealthCheckType string `choices:"tcp|http"`

	HealthCheckDomain   string
	HealthCheckURI      string
	HealthCheckHttpCode string

	HealthCheckRise     *int
	HealthCheckFall     *int
	HealthCheckInterval *int
	HealthCheckTimeout  *int

	HealthCheckReq string
	HealthCheckExp string

	StickySession              string
	StickySessionType          string
	StickySessionCookie        string
	StickySessionCookieTimeout *int

	XForwardedFor string `choices:"true|false"`
	Gzip          string `choices:"true|false"`

	Certificate     string
	TLSCipherPolicy string
	EnableHttp2     string `choices:"true|false"`

	HTTPRequestRate       *int
	HTTPRequestRatePerSrc *int
}

type LoadbalancerListenerGetOptions struct {
	ID string
}

type LoadbalancerListenerDeleteOptions struct {
	ID string
}

type LoadbalancerListenerActionStatusOptions struct {
	ID     string
	Status string `choices:"enabled|disabled"`
}

type LoadbalancerListenerActionSyncStatusOptions struct {
	ID string
}
