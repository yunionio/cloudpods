// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"time"
)

type Loadbalancer struct {
	VirtualResource
	ManagedResource

	EgressMbps  int
	Address     string
	AddressType string
	NetworkType string
	NetworkId   string
	VpcId       string
	ZoneId      string
	ClusterId   string

	BackendGroupId   string
	CloudregionId    string
	ChargeType       string
	LoadbalancerSpec string
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

type LoadbalancerHTTPRateLimiter struct {
	HTTPRequestRate       int
	HTTPRequestRatePerSrc int
}

type LoadbalancerListener struct {
	VirtualResource
	ManagedResource

	CloudregionId  string
	LoadbalancerId string
	ListenerType   string
	ListenerPort   int
	EgressMbps     int

	Scheduler string

	SendProxy string

	ClientRequestTimeout  int
	ClientIdleTimeout     int
	BackendConnectTimeout int
	BackendIdleTimeout    int

	BackendGroupId    string
	BackendServerPort int

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

	LoadbalancerHTTPRateLimiter
}

type LoadbalancerListenerRule struct {
	VirtualResource
	ManagedResource

	CloudregionId  string
	ListenerId     string
	BackendGroupId string

	Domain string
	Path   string

	LoadbalancerHTTPRateLimiter
}

type LoadbalancerBackendGroup struct {
	VirtualResource
	ManagedResource

	Type           string
	LoadbalancerId string
	CloudregionId  string
}

type LoadbalancerBackend struct {
	VirtualResource
	ManagedResource

	CloudregionId  string
	BackendGroupId string
	BackendId      string
	BackendType    string
	BackendRole    string
	Weight         int
	Address        string
	Port           int

	SendProxy string
}

type LoadbalancerAclEntry struct {
	Cidr    string
	Comment string
}
type LoadbalancerAclEntries []*LoadbalancerAclEntry
type LoadbalancerAcl struct {
	SharableVirtualResource
	ManagedResource

	AclEntries    *LoadbalancerAclEntries
	CloudregionId string
}

type LoadbalancerCertificate struct {
	VirtualResource
	ManagedResource

	Certificate string
	PrivateKey  string

	CloudregionId           string
	PublicKeyAlgorithm      string
	PublicKeyBitLen         int
	SignatureAlgorithm      string
	Fingerprint             string
	NotBefore               time.Time
	NotAfter                time.Time
	CommonName              string
	SubjectAlternativeNames string
}

type LoadbalancerCluster struct {
	StandaloneResource
	ZoneId string
}

type LoadbalancerAgent struct {
	StandaloneResource

	Version    string
	IP         string
	HaState    string
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

	ClusterId  string
	Deployment LoadbalancerDeployment
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
	InfluxDbOutputUrl       string
	InfluxDbOutputName      string
	InfluxDbOutputUnsafeSsl bool
	HaproxyInputInterval    int
}

type LoadbalancerAgentParams struct {
	KeepalivedConfTmpl string
	HaproxyConfTmpl    string
	TelegrafConfTmpl   string
	Vrrp               LoadbalancerAgentParamsVrrp
	Haproxy            LoadbalancerAgentParamsHaproxy
	Telegraf           LoadbalancerAgentParamsTelegraf
}

type LoadbalancerDeployment struct {
	Host            string
	AnsiblePlaybook string
}
