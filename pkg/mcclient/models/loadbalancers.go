package models

import (
	"time"
)

type Loadbalancer struct {
	VirtualResource

	Address     string
	AddressType string
	NetworkType string
	NetworkId   string
	ZoneId      string

	BackendGroupId string
}

type LoadbalancerTCPListener struct{}
type LoadbalancerUDPListener struct{}

type LoadbalancerHTTPListener struct {
	StickySession              string
	StickySessionType          string
	StickySessionCookie        string
	StickySessionCookieTimeout int

	XForwardedFor bool
	Gzip          bool
}

// CACertificate string
type LoadbalancerHTTPSListener struct {
	CertificateId   string
	TLSCipherPolicy string
	EnableHttp2     bool
}

type LoadbalancerListener struct {
	VirtualResource

	LoadbalancerId string
	ListenerType   string
	ListenerPort   int

	Scheduler string

	ClientRequestTimeout  int
	ClientIdleTimeout     int
	BackendConnectTimeout int
	BackendIdleTimeout    int

	BackendGroupId string

	AclStatus string
	AclType   string
	AclId     string

	HealthCheck     string
	HealthCheckType string

	HealthCheckDomain   string
	HealthCheckURI      string
	HealthCheckHttpCode string

	HealthCheckRise     int
	HealthCheckFall     int
	HealthCheckInterval int
	HealthCheckTimeout  int

	HealthCheckReq string
	HealthCheckExp string

	LoadbalancerTCPListener
	LoadbalancerUDPListener
	LoadbalancerHTTPListener
	LoadbalancerHTTPSListener
}

type LoadbalancerListenerRule struct {
	VirtualResource

	ListenerId     string
	BackendGroupId string

	Domain string
	Path   string
}

type LoadbalancerBackendGroup struct {
	VirtualResource

	LoadbalancerId string
}

type LoadbalancerBackend struct {
	VirtualResource

	BackendGroupId string
	BackendId      string
	BackendType    string
	Weight         int
	Address        string
	Port           int
}

type LoadbalancerAclEntry struct {
	Cidr    string
	Comment string
}
type LoadbalancerAclEntries []*LoadbalancerAclEntry
type LoadbalancerAcl struct {
	SharableVirtualResource

	AclEntries *LoadbalancerAclEntries
}

type LoadbalancerCertificate struct {
	VirtualResource

	Certificate string
	PrivateKey  string

	PublicKeyAlgorithm      string
	PublicKeyBitLen         int
	SignatureAlgorithm      string
	FingerprintSha256       string
	NotBefore               time.Time
	NotAfter                time.Time
	CommonName              string
	SubjectAlternativeNames string
}

type LoadbalancerAgent struct {
	StandaloneResource

	HbLastSeen time.Time
	HbTimeout  int

	Loadbalancers             time.Time
	LoadbalancerListeners     time.Time
	LoadbalancerListenerRules time.Time
	LoadbalancerBackendGroups time.Time
	LoadbalancerBackends      time.Time
	LoadbalancerAcls          time.Time
	LoadbalancerCertificates  time.Time
	Params                    LoadbalancerAgentParams
}

type LoadbalancerAgentParamsVrrp struct {
	Priority          int
	VirtualRouterId   int
	GarpMasterRefresh int
	Preempt           bool
	Interface         string
	AdvertInt         int
	Pass              string
}

type LoadbalancerAgentParamsHaproxy struct {
	GlobalLog      string
	GlobalNbthread int
	LogHttp        bool
	LogTcp         bool
	LogNormal      bool
}

type LoadbalancerAgentParamsTelegraf struct {
	InfluxDbOutputUrl    string
	InfluxDbOutputName   string
	HaproxyInputInterval int
}

type LoadbalancerAgentParams struct {
	KeepalivedConfTmpl string
	HaproxyConfTmpl    string
	TelegrafConfTmpl   string
	Vrrp               LoadbalancerAgentParamsVrrp
	Haproxy            LoadbalancerAgentParamsHaproxy
	Telegraf           LoadbalancerAgentParamsTelegraf
}
