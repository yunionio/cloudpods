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

package options

type LoadbalancerListenerCreateOptions struct {
	NAME string

	Loadbalancer      string `required:"true"`
	ListenerType      string `required:"true" choices:"tcp|udp|http|https"`
	ListenerPort      *int   `required:"true"`
	BackendServerPort *int
	BackendGroup      string

	Scheduler string `required:"true" choices:"rr|wrr|wlc|sch|tch"`

	SendProxy string `choices:"off|v1|v2|v2-ssl|v2-ssl-cn"`

	ClientRequestTimeout  *int
	ClientIdleTimeout     *int
	BackendConnectTimeout *int
	BackendIdleTimeout    *int

	AclStatus string `choices:"on|off"`
	AclType   string `choices:"black|white"`
	Acl       string

	EgressMbps int

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

	SendProxy string `choices:"off|v1|v2|v2-ssl|v2-ssl-cn"`

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
	ID   string `json:"-"`
	Name string

	BackendGroup string

	Scheduler string `choices:"rr|wrr|wlc|sch|tch"`

	SendProxy string `choices:"off|v1|v2|v2-ssl|v2-ssl-cn"`

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
	ID string `json:"-"`
}

type LoadbalancerListenerDeleteOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListenerActionStatusOptions struct {
	ID     string `json:"-"`
	Status string `choices:"enabled|disabled"`
}

type LoadbalancerListenerGetBackendStatusOptions struct {
	ID string `json:"-"`
}

type LoadbalancerListenerActionSyncStatusOptions struct {
	ID string `json:"-"`
}
