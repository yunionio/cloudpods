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

package compute

import (
	"regexp"
	"strings"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type LoadbalancerListenerDetails struct {
	apis.StatusStandaloneResourceDetails
	LoadbalancerResourceInfo
	LoadbalancerAclResourceInfo
	LoadbalancerCertificateResourceInfo

	SLoadbalancerListener

	BackendGroup    string `json:"backend_group"`
	CertificateName string `json:"certificate_name"`

	ProjectId string `json:"tenant_id"`
}

type LoadbalancerListenerResourceInfo struct {
	// 负载均衡监听器名称
	Listener string `json:"listener"`

	// 负载均衡ID
	LoadbalancerId string `json:"loadbalancer_id"`

	LoadbalancerResourceInfo
}

type LoadbalancerListenerResourceInput struct {
	// 负载均衡监听器
	ListenerId string `json:"listener_id"`

	// 负载均衡监听器ID
	// swagger:ignore
	// Deprecated
	Listener string `json:"listener" yunion-deprecated-by:"listener_id"`
}

type LoadbalancerListenerFilterListInput struct {
	LoadbalancerFilterListInput

	LoadbalancerListenerResourceInput

	// 以负载均衡监听器名称排序
	OrderByListener string `json:"order_by_listener"`
}

type LoadbalancerListenerListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.ExternalizedResourceBaseListInput
	LoadbalancerFilterListInput
	// filter by acl
	LoadbalancerAclResourceInput

	// filter by backend_group
	BackendGroup string `json:"backend_group"`

	ListenerType []string `json:"listener_type"`
	ListenerPort []int    `json:"listener_port"`

	Scheduler []string `json:"scheduler"`

	Certificate []string `json:"certificate_id"`

	SendProxy []string `json:"send_proxy"`

	AclStatus []string `json:"acl_status"`
	AclType   []string `json:"acl_type"`
}

type LoadbalancerListenerCreateInput struct {
	apis.StatusStandaloneResourceCreateInput

	// swagger: ignore
	Loadbalancer string `json:"loadbalancer" yunion-deprecated-by:"loadbalancer_id"`
	// 负载均衡ID
	LoadbalancerId string `json:"loadbalancer_id"`

	//swagger: ignore
	BackendGroup   string `json:"backend_group" yunion-deprecated-by:"backend_group_id"`
	BackendGroupId string `json:"backend_group_id"`

	// range 1-600
	// default: 10
	ClientReqeustTimeout int `json:"client_request_timeout"`
	// range 1-600
	// default: 90
	ClientIdleTimeout int `json:"client_idle_timeout"`
	// range 1-180
	// default: 5
	BackendConnectTimeout int `json:"backend_connect_timeout"`
	// range 1-600
	// default: 90
	BackendIdleTimeout int `json:"backend_idle_timeout"`

	// required: true
	// enmu: tcp, udp, http, https
	ListenerType               string `json:"listener_type"`
	ListenerPort               int    `json:"listener_port"`
	SendProxy                  string `json:"send_proxy"`
	Scheduler                  string `json:"scheduler"`
	StickySession              string `json:"sticky_session"`
	StickySessionType          string `json:"sticky_session_type"`
	StickySessionCookie        string `json:"sticky_session_cookie"`
	StickySessionCookieTimeout int    `json:"sticky_session_cookie_timeout"`
	// default: true
	XForwardedFor *bool `json:"x_forwarded_for"`
	Gzip          bool  `json:"gzip"`

	EgressMbps int `json:"egress_mbps"`

	//swagger: ignore
	Certificate   string `json:"certificate" yunion-deprecated-by:"certificate_id"`
	CertificateId string `json:"certificate_id"`

	TLSCipherPolicy string `json:"tls_cipher_policy"`
	// default: true
	EnableHttp2 *bool `json:"enable_http2"`

	// default: on
	// enmu: on, off
	HealthCheck string `json:"health_check"`
	// enum: ["tcp", "udp", "http"]
	HealthCheckType   string `json:"health_check_type"`
	HealthCheckDomain string `json:"health_check_domain"`
	HealthCheckPath   string `json:"health_check_path"`
	// default: http_2xx,http_3xx
	HealthCheckHttpCode string `json:"health_check_http_code"`

	// 2-10
	// default: 5
	HealthCheckRise int `json:"health_check_rise"`
	// 2-10
	// default: 2
	HealthCheckFail int `json:"health_check_fail"`
	// 2-120
	// default: 5
	HealthCheckTimeout int `json:"health_check_timeout"`
	// 5-300
	// default: 30
	HealthCheckInterval int `json:"health_check_interval"`

	HttpRequestRate       int `json:"http_request_rate"`
	HttpRequestRatePerSrc int `json:"http_request_rate_src"`

	// default: off
	// enmu: off, raw
	Redirect       string `json:"redirect"`
	RedirectCode   int    `json:"redirect_code"`
	RedirectScheme string `json:"redirect_scheme"`
	RedirectHost   string `json:"redirect_host"`
	RedirectPath   string `json:"redirect_path"`

	//swagger: ignore
	Acl string `json:"acl" yunion-deprecated-by:"acl_id"`

	AclId     string `json:"acl_id"`
	AclStatus string `json:"acl_status"`
	AclType   string `json:"acl_type"`
}

func (self *LoadbalancerListenerCreateInput) Validate() error {
	if len(self.Status) == 0 {
		self.Status = LB_STATUS_ENABLED
	}
	if len(self.SendProxy) == 0 {
		self.SendProxy = LB_SENDPROXY_OFF
	}
	if !utils.IsInStringArray(self.SendProxy, LB_SENDPROXY_CHOICES) {
		return httperrors.NewInputParameterError("invalid send_proxy %s", self.SendProxy)
	}
	if !utils.IsInStringArray(self.Scheduler, LB_SCHEDULER_TYPES) {
		return httperrors.NewInputParameterError("invalid scheduler %s", self.Scheduler)
	}
	if len(self.StickySession) == 0 {
		self.StickySession = LB_BOOL_OFF
	}
	switch self.StickySession {
	case LB_BOOL_OFF:
	case LB_BOOL_ON:
		if len(self.StickySessionType) == 0 {
			self.StickySessionType = LB_STICKY_SESSION_TYPE_INSERT
		}
		if !utils.IsInStringArray(self.StickySessionType, LB_STICKY_SESSION_TYPES) {
			return httperrors.NewInputParameterError("invalid sticky_session_type %s", self.StickySessionType)
		}
		reg := regexp.MustCompile(`\w+`)
		if len(self.StickySessionCookie) > 0 && !reg.MatchString(self.StickySessionCookie) {
			return httperrors.NewInputParameterError("invalid sticky_session_cookie %s", self.StickySessionCookie)
		}
	default:
		return httperrors.NewInputParameterError("invalid sticky_session %s", self.StickySession)
	}
	if self.XForwardedFor == nil {
		forward := true
		self.XForwardedFor = &forward
	}
	if len(self.ListenerType) == 0 {
		return httperrors.NewMissingParameterError("listener_type")
	}
	if !utils.IsInStringArray(self.ListenerType, LB_LISTENER_TYPES) {
		return httperrors.NewInputParameterError("invalid listener_type %s", self.ListenerType)
	}
	if self.ListenerPort < 1 || self.ListenerPort > 65535 {
		return httperrors.NewOutOfRangeError("listener_port out of range 1-65535")
	}
	if self.ListenerType == LB_LISTENER_TYPE_HTTPS {
		if len(self.TLSCipherPolicy) == 0 {
			self.TLSCipherPolicy = LB_TLS_CIPHER_POLICY_1_2
		}
		if !utils.IsInStringArray(self.TLSCipherPolicy, LB_TLS_CIPHER_POLICIES) {
			return httperrors.NewInputParameterError("invalid tls_cipher_policy %s", self.TLSCipherPolicy)
		}
		if self.EnableHttp2 == nil {
			enabled := true
			self.EnableHttp2 = &enabled
		}
	}
	if len(self.HealthCheck) == 0 {
		self.HealthCheck = LB_BOOL_ON
	}
	if !utils.IsInStringArray(self.HealthCheck, []string{LB_BOOL_ON, LB_BOOL_OFF}) {
		return httperrors.NewInputParameterError("invalid health_check %s", self.HealthCheck)
	}
	if self.HealthCheck == LB_BOOL_ON {
		if self.ListenerType == LB_LISTENER_TYPE_HTTPS && len(self.HealthCheckType) == 0 {
			self.HealthCheckType = LB_HEALTH_CHECK_HTTP
		}
		if !utils.IsInStringArray(self.HealthCheckType, LB_HEALTH_CHECK_TYPES) {
			return httperrors.NewInputParameterError("invalid health_check_type %s", self.HealthCheckType)
		}
		if len(self.HealthCheckHttpCode) == 0 {
			self.HealthCheckHttpCode = LB_HEALTH_CHECK_HTTP_CODE_DEFAULT
		}
		for _, code := range strings.Split(self.HealthCheckHttpCode, ",") {
			if !utils.IsInStringArray(code, LB_HEALTH_CHECK_HTTP_CODES) {
				return httperrors.NewInputParameterError("invalid health_check_http_code: %s", code)
			}
		}
		if self.HealthCheckRise < 2 || self.HealthCheckRise > 10 {
			self.HealthCheckRise = 5
		}
		if self.HealthCheckFail < 2 || self.HealthCheckFail > 10 {
			self.HealthCheckFail = 2
		}
		if self.HealthCheckTimeout < 2 || self.HealthCheckTimeout > 120 {
			self.HealthCheckTimeout = 5
		}
		if self.HealthCheckInterval < 5 || self.HealthCheckInterval > 300 {
			self.HealthCheckInterval = 30
		}
	}
	if len(self.Redirect) == 0 {
		self.Redirect = LB_REDIRECT_OFF
	}
	if !utils.IsInStringArray(self.Redirect, []string{LB_REDIRECT_OFF, LB_REDIRECT_RAW}) {
		return httperrors.NewInputParameterError("invalid redirect %s", self.Redirect)
	}
	return nil
}

type LoadbalancerListenerUpdateInput struct {
	apis.StatusStandaloneResourceBaseUpdateInput

	AclStatus *string `json:"acl_status"`
	AclType   *string `json:"acl_type"`
	//swagger: ignore
	Acl   *string `json:"acl" yunion-deprecated-by:"acl_id"`
	AclId *string `json:"acl_id"`

	// range 1-600
	// default: 10
	ClientReqeustTimeout *int `json:"client_request_timeout"`
	// range 1-600
	// default: 90
	ClientIdleTimeout *int `json:"client_idle_timeout"`
	// range 1-180
	// default: 5
	BackendConnectTimeout *int `json:"backend_connect_timeout"`
	// range 1-600
	// default: 90
	BackendIdleTimeout *int `json:"backend_idle_timeout"`

	SendProxy                  *string `json:"send_proxy"`
	Scheduler                  *string `json:"scheduler"`
	StickySession              *string `json:"sticky_session"`
	StickySessionType          *string `json:"sticky_session_type"`
	StickySessionCookie        *string `json:"sticky_session_cookie"`
	StickySessionCookieTimeout *int    `json:"sticky_session_cookie_timeout"`
	// default: true
	XForwardedFor *bool `json:"x_forwarded_for"`
	Gzip          *bool `json:"gzip"`

	//swagger: ignore
	Certificate   *string `json:"certificate" yunion-deprecated-by:"certificate_id"`
	CertificateId *string `json:"certificate_id"`

	TLSCipherPolicy *string `json:"tls_cipher_policy"`
	// default: true
	EnableHttp2 *bool `json:"enable_http2"`

	// default: on
	// enmu: on, off
	HealthCheck *string `json:"health_check"`
	// enum: ["tcp", "udp", "http"]
	HealthCheckType   *string `json:"health_check_type"`
	HealthCheckDomain *string `json:"string"`
	HealthCheckPath   *string `json:"health_check_path"`
	// default: http_2xx,http_3xx
	HealthCheckHttpCode *string `json:"health_check_http_code"`

	// 2-10
	// default: 5
	HealthCheckRise *int `json:"health_check_rise"`
	// 2-10
	// default: 2
	HealthCheckFail *int `json:"health_check_fail"`
	// 2-120
	// default: 5
	HealthCheckTimeout *int `json:"health_check_timeout"`
	// 5-300
	// default: 30
	HealthCheckInterval *int `json:"health_check_interval"`

	HttpRequestRate       *int `json:"http_request_rate"`
	HttpRequestRatePerSrc *int `json:"http_request_rate_src"`

	// default: off
	// enmu: off, raw
	Redirect       *string `json:"redirect"`
	RedirectCode   *int    `json:"redirect_code"`
	RedirectScheme *string `json:"redirect_scheme"`
	RedirectHost   *string `json:"redirect_host"`
	RedirectPath   *string `json:"redirect_path"`
}

func (self *LoadbalancerListenerUpdateInput) Validate() error {
	if self.AclStatus != nil {
		if !utils.IsInStringArray(*self.AclStatus, []string{LB_BOOL_ON, LB_BOOL_OFF}) {
			return httperrors.NewInputParameterError("invalid acl_status %v", self.AclStatus)
		}
		if *self.AclStatus == LB_BOOL_ON {
			if self.AclType == nil {
				return httperrors.NewMissingParameterError("acl_type")
			}
			if !utils.IsInStringArray(*self.AclType, LB_ACL_TYPES) {
				return httperrors.NewInputParameterError("invalid acl_type %v", self.AclType)
			}
		}
	}
	if self.Scheduler != nil && !utils.IsInStringArray(*self.Scheduler, LB_SCHEDULER_TYPES) {
		return httperrors.NewInputParameterError("invalid scheduler %v", self.Scheduler)
	}
	if self.TLSCipherPolicy != nil && !utils.IsInStringArray(*self.TLSCipherPolicy, LB_TLS_CIPHER_POLICIES) {
		return httperrors.NewInputParameterError("invalid tls_cipher_policy %v", self.TLSCipherPolicy)
	}
	if self.SendProxy != nil && !utils.IsInStringArray(*self.SendProxy, LB_SENDPROXY_CHOICES) {
		return httperrors.NewInputParameterError("invalid send_proxy %v", self.SendProxy)
	}
	if self.StickySession != nil {
		if !utils.IsInStringArray(*self.StickySession, []string{LB_BOOL_ON, LB_BOOL_OFF}) {
			return httperrors.NewInputParameterError("invalid sticky_session %v", self.StickySession)
		}
		if *self.StickySession == LB_BOOL_ON {
			if self.StickySessionType == nil {
				return httperrors.NewMissingParameterError("sticky_session_type")
			}
			if !utils.IsInStringArray(*self.StickySessionType, LB_STICKY_SESSION_TYPES) {
				return httperrors.NewInputParameterError("invalid sticky_session_type %v", self.StickySessionType)
			}
		}
	}
	if self.HealthCheck != nil {
		if !utils.IsInStringArray(*self.HealthCheck, []string{LB_BOOL_ON, LB_BOOL_OFF}) {
			return httperrors.NewInputParameterError("invalid health_check %v", self.HealthCheck)
		}
		if *self.HealthCheck == LB_BOOL_ON {
			if self.HealthCheckType == nil {
				return httperrors.NewMissingParameterError("health_check_type")
			}
			if !utils.IsInStringArray(*self.HealthCheckType, LB_HEALTH_CHECK_TYPES) {
				return httperrors.NewInputParameterError("invalid health_cheack_type %v", self.HealthCheckType)
			}
			checkCode := LB_HEALTH_CHECK_HTTP_CODE_DEFAULT
			if self.HealthCheckHttpCode == nil {
				self.HealthCheckHttpCode = &checkCode
			}
			for _, code := range strings.Split(*self.HealthCheckHttpCode, ",") {
				if !utils.IsInStringArray(code, LB_HEALTH_CHECK_HTTP_CODES) {
					return httperrors.NewInputParameterError("invalid health_check_http_code: %s", code)
				}
			}
			if self.HealthCheckRise == nil || *self.HealthCheckRise < 2 || *self.HealthCheckRise > 10 {
				rise := 5
				self.HealthCheckRise = &rise
			}
			if self.HealthCheckFail == nil || *self.HealthCheckFail < 2 || *self.HealthCheckFail > 10 {
				fail := 2
				self.HealthCheckFail = &fail
				if self.HealthCheckTimeout == nil || *self.HealthCheckTimeout < 2 || *self.HealthCheckTimeout > 120 {
					timeout := 5
					self.HealthCheckTimeout = &timeout
				}
				if self.HealthCheckInterval == nil || *self.HealthCheckInterval < 5 || *self.HealthCheckInterval > 300 {
					interval := 30
					self.HealthCheckInterval = &interval
				}
			}
		}
	}
	return nil
}
