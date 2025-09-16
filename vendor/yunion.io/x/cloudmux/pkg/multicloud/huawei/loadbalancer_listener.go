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

package huawei

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type InsertHeaders struct {
	XForwardedELBIP bool `json:"X-Forwarded-ELB-IP"`
}

type Loadbalancer struct {
	ID string `json:"id"`
}

type SElbListener struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	HuaweiTags
	lb           *SLoadbalancer
	acl          *SElbACL
	backendgroup *SElbBackendGroup

	ProtocolPort           int            `json:"protocol_port"`
	Protocol               string         `json:"protocol"`
	Description            string         `json:"description"`
	AdminStateUp           bool           `json:"admin_state_up"`
	Http2Enable            bool           `json:"http2_enable"`
	Loadbalancers          []Loadbalancer `json:"loadbalancers"`
	TenantID               string         `json:"tenant_id"`
	ProjectID              string         `json:"project_id"`
	ConnectionLimit        int            `json:"connection_limit"`
	DefaultPoolID          string         `json:"default_pool_id"`
	ID                     string         `json:"id"`
	Name                   string         `json:"name"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	InsertHeaders          InsertHeaders  `json:"insert_headers"`
	DefaultTlsContainerRef string         `json:"default_tls_container_ref"`
}

func (self *SElbListener) GetId() string {
	return self.ID
}

func (self *SElbListener) GetName() string {
	return self.Name
}

func (self *SElbListener) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbListener) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbListener) Refresh() error {
	ilistener, err := self.lb.GetILoadBalancerListenerById(self.GetId())
	if err != nil {
		return err
	}

	listener := ilistener.(*SElbListener)
	listener.lb = self.lb
	err = jsonutils.Update(self, listener)
	if err != nil {
		return err
	}

	return nil
}

func (self *SElbListener) IsEmulated() bool {
	return false
}

func (self *SElbListener) GetProjectId() string {
	return self.ProjectID
}

func (self *SElbListener) GetListenerType() string {
	switch self.Protocol {
	case "TCP":
		return api.LB_LISTENER_TYPE_TCP
	case "UDP":
		return api.LB_LISTENER_TYPE_UDP
	case "HTTP":
		return api.LB_LISTENER_TYPE_HTTP
	case "TERMINATED_HTTPS":
		return api.LB_LISTENER_TYPE_HTTPS
	case "HTTPS":
		return api.LB_LISTENER_TYPE_HTTPS
	default:
		return ""
	}
}

func (self *SElbListener) GetListenerPort() int {
	return self.ProtocolPort
}

func (self *SElbListener) GetBackendGroup() (*SElbBackendGroup, error) {
	if self.backendgroup == nil {
		lbbgId := self.GetBackendGroupId()
		if len(lbbgId) > 0 {
			lbbg, err := self.lb.GetILoadBalancerBackendGroupById(lbbgId)
			if err != nil {
				return nil, err
			}

			self.backendgroup = lbbg.(*SElbBackendGroup)
		}
	}

	return self.backendgroup, nil
}

func (self *SElbListener) GetScheduler() string {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetScheduler %s", err.Error())
	}

	if lbbg == nil {
		return ""
	}

	return lbbg.GetScheduler()
}

func (self *SElbListener) GetAcl() (*SElbACL, error) {
	if self.acl != nil {
		return self.acl, nil
	}

	acls, err := self.lb.region.GetLoadBalancerAcls(self.GetId())
	if err != nil {
		return nil, err
	}

	if len(acls) == 0 {
		return nil, nil
	} else {
		self.acl = &acls[0]
		return &acls[0], nil
	}
}

func (self *SElbListener) GetAclStatus() string {
	acl, err := self.GetAcl()
	if err != nil {
		log.Debugf("GetAclStatus %s", err)
		return ""
	}

	if acl != nil && acl.EnableWhitelist {
		return api.LB_BOOL_ON
	}

	return api.LB_BOOL_OFF
}

func (self *SElbListener) GetAclType() string {
	return api.LB_ACL_TYPE_WHITE
}

func (self *SElbListener) GetAclId() string {
	acl, err := self.GetAcl()
	if err != nil {
		log.Debugf("GetAclStatus %s", err)
		return ""
	}

	if acl == nil {
		return ""
	}

	return acl.GetId()
}

func (self *SElbListener) GetEgressMbps() int {
	return 0
}

func (self *SElbListener) GetHealthCheck() string {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheck %s", err.Error())
	}

	if lbbg == nil {
		return ""
	}

	health, err := lbbg.GetHealthCheck()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheck %s", err.Error())
	}

	if health != nil {
		return api.LB_BOOL_ON
	} else {
		return api.LB_BOOL_OFF
	}
}

func (self *SElbListener) GetHealthCheckType() string {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckType %s", err.Error())
	}

	if lbbg == nil {
		return ""
	}

	health, err := lbbg.GetHealthCheck()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckType %s", err.Error())
	}

	if health != nil {
		return health.HealthCheckType
	}

	return ""
}

func (self *SElbListener) GetHealthCheckTimeout() int {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckTimeout %s", err.Error())
	}

	if lbbg == nil {
		return 0
	}

	health, err := lbbg.GetHealthCheck()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckTimeout %s", err.Error())
	}

	if health != nil {
		return health.HealthCheckTimeout
	}

	return 0
}

func (self *SElbListener) GetHealthCheckInterval() int {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckInterval %s", err.Error())
	}

	if lbbg == nil {
		return 0
	}

	health, err := lbbg.GetHealthCheck()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckInterval %s", err.Error())
	}

	if health != nil {
		return health.HealthCheckInterval
	}

	return 0
}

func (self *SElbListener) GetHealthCheckRise() int {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckRise %s", err.Error())
	}

	if lbbg == nil {
		return 0
	}

	health, err := lbbg.GetHealthCheck()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckRise %s", err.Error())
	}

	if health != nil {
		return health.HealthCheckRise
	} else {
		return 0
	}
}

func (self *SElbListener) GetHealthCheckFail() int {
	return 0
}

func (self *SElbListener) GetHealthCheckReq() string {
	return ""
}

func (self *SElbListener) GetHealthCheckExp() string {
	return ""
}

func (self *SElbListener) GetBackendGroupId() string {
	return self.DefaultPoolID
}

func (self *SElbListener) GetBackendServerPort() int {
	return 0
}

func (self *SElbListener) GetHealthCheckDomain() string {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckDomain %s", err.Error())
	}

	if lbbg == nil {
		return ""
	}

	health, err := lbbg.GetHealthCheck()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckDomain %s", err.Error())
	}

	if health != nil {
		return health.HealthCheckDomain
	}

	return ""
}

func (self *SElbListener) GetHealthCheckURI() string {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckURI %s", err.Error())
	}

	if lbbg == nil {
		return ""
	}

	health, err := lbbg.GetHealthCheck()
	if err != nil {
		log.Errorf("ElbListener GetHealthCheckURI %s", err.Error())
	}

	if health != nil {
		return health.HealthCheckURI
	}

	return ""
}

func (self *SElbListener) ChangeCertificate(ctx context.Context, opts *cloudprovider.ListenerCertificateOptions) error {
	params := map[string]interface{}{
		"default_tls_container_ref": opts.CertificateId,
	}
	_, err := self.lb.region.put(SERVICE_ELB, "elb/listeners/"+self.ID, map[string]interface{}{"listener": params})
	return err
}

func (listerner *SElbListener) SetAcl(ctx context.Context, opts *cloudprovider.ListenerAclOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElbListener) GetHealthCheckCode() string {
	return ""
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0136295317.html
func (self *SElbListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	l7policy, err := self.lb.region.CreateLoadBalancerPolicy(self.GetId(), rule)
	if err != nil {
		return nil, err
	}

	l7policy.region = self.lb.region
	l7policy.lb = self.lb
	l7policy.listener = self
	return l7policy, nil
}

func (self *SElbListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	ret := &SElbListenerPolicy{region: self.lb.region, lb: self.lb, listener: self}
	resp, err := self.lb.region.list(SERVICE_ELB, "elb/l7policies/"+ruleId, nil)
	if err != nil {
		return nil, err
	}
	return ret, resp.Unmarshal(ret, "l7policy")
}

func (self *SElbListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	ret, err := self.lb.region.GetLoadBalancerPolicies(self.GetId())
	if err != nil {
		return nil, err
	}

	iret := []cloudprovider.ICloudLoadbalancerListenerRule{}
	for i := range ret {
		rule := ret[i]
		rule.listener = self
		rule.lb = self.lb
		rule.region = self.lb.region
		iret = append(iret, &rule)
	}

	return iret, nil
}

func (self *SElbListener) GetStickySession() string {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetStickySession %s", err.Error())
	}

	if lbbg == nil {
		return ""
	}

	stickySession, err := lbbg.GetStickySession()
	if err != nil {
		log.Errorf("ElbListener GetStickySession %s", err.Error())
	}

	if stickySession != nil {
		return stickySession.StickySession
	}

	return ""
}

func (self *SElbListener) GetStickySessionType() string {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetStickySessionType %s", err.Error())
	}

	if lbbg == nil {
		return ""
	}

	stickySession, err := lbbg.GetStickySession()
	if err != nil {
		log.Errorf("ElbListener GetStickySessionType %s", err.Error())
	}

	if stickySession != nil {
		return stickySession.StickySessionType
	}

	return ""
}

func (self *SElbListener) GetStickySessionCookie() string {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetStickySessionCookie %s", err.Error())
	}

	if lbbg == nil {
		return ""
	}

	stickySession, err := lbbg.GetStickySession()
	if err != nil {
		log.Errorf("ElbListener GetStickySessionCookie %s", err.Error())
	}

	if stickySession != nil {
		return stickySession.StickySessionCookie
	}

	return ""
}

func (self *SElbListener) GetStickySessionCookieTimeout() int {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		log.Errorf("ElbListener GetStickySessionCookieTimeout %s", err.Error())
	}

	if lbbg == nil {
		return 0
	}

	stickySession, err := lbbg.GetStickySession()
	if err != nil {
		log.Errorf("ElbListener GetStickySessionCookieTimeout %s", err.Error())
	}

	if stickySession != nil {
		return stickySession.StickySessionCookieTimeout
	}

	return 0
}

func (self *SElbListener) XForwardedForEnabled() bool {
	return self.InsertHeaders.XForwardedELBIP
}

func (self *SElbListener) GzipEnabled() bool {
	return false
}

func (self *SElbListener) GetCertificateId() string {
	return self.DefaultTlsContainerRef
}

func (self *SElbListener) GetTLSCipherPolicy() string {
	return ""
}

func (self *SElbListener) HTTP2Enabled() bool {
	return self.Http2Enable
}

func (self *SElbListener) Start() error {
	return nil
}

func (self *SElbListener) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElbListener) ChangeScheduler(ctx context.Context, opts *cloudprovider.ChangeListenerSchedulerOptions) error {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		return errors.Wrapf(err, "GetBackendGroup")
	}
	pool := map[string]interface{}{
		"session_persistence": jsonutils.JSONNull,
	}
	switch opts.Scheduler {
	case api.LB_SCHEDULER_WRR:
		pool["lb_algorithm"] = "ROUND_ROBIN"
	case api.LB_SCHEDULER_WLC:
		pool["lb_algorithm"] = "LEAST_CONNECTIONS"
	case api.LB_SCHEDULER_SCH:
		pool["lb_algorithm"] = "SOURCE_IP"
	default:
		return errors.Wrapf(cloudprovider.ErrNotSupported, "invalid scheduler %s", opts.Scheduler)
	}
	if opts.StickySession == api.LB_BOOL_ON {
		sticky := map[string]interface{}{
			"type":                "SOURCE_IP",
			"persistence_timeout": opts.StickySessionCookieTimeout,
		}
		if opts.StickySessionType == api.LB_STICKY_SESSION_TYPE_SERVER {
			sticky["type"] = "APP_COOKIE"
			sticky["cookie_name"] = opts.StickySessionCookie
		}
		pool["session_persistence"] = sticky
	}
	params := map[string]interface{}{
		"pool": pool,
	}
	_, err = self.lb.region.put(SERVICE_ELB, fmt.Sprintf("elb/pools/%s", lbbg.ID), params)
	return err
}

func (self *SElbListener) SetHealthCheck(ctx context.Context, opts *cloudprovider.ListenerHealthCheckOptions) error {
	lbbg, err := self.GetBackendGroup()
	if err != nil {
		return errors.Wrapf(err, "GetBackendGroup")
	}
	if opts.HealthCheck == api.LB_BOOL_OFF {
		if len(lbbg.HealthMonitorID) > 0 {
			return self.lb.region.DeleteLoadbalancerHealthCheck(lbbg.HealthMonitorID)
		}
		return nil
	}
	if opts.HealthCheckType == api.LB_HEALTH_CHECK_UDP {
		opts.HealthCheckType = "UDP_CONNECT"
	}
	if opts.HealthCheckRise < 1 || opts.HealthCheckRise > 10 {
		opts.HealthCheckRise = 3
	}
	heathmonitor := map[string]interface{}{
		"delay":       opts.HealthCheckInterval,
		"max_retries": opts.HealthCheckRise,
		"timeout":     opts.HealthCheckTimeout,
		"type":        strings.ToUpper(opts.HealthCheckType),
	}
	if opts.HealthCheckType == api.LB_HEALTH_CHECK_HTTP {
		if len(opts.HealthCheckDomain) > 0 {
			heathmonitor["domain_name"] = opts.HealthCheckDomain
		}
		heathmonitor["url_path"] = opts.HealthCheckURI
	}
	if len(lbbg.HealthMonitorID) == 0 {
		heathmonitor["pool_id"] = lbbg.ID
		params := map[string]interface{}{
			"healthmonitor": heathmonitor,
		}
		_, err = self.lb.region.post(SERVICE_ELB, "elb/healthmonitors", params)
		return err
	}
	params := map[string]interface{}{
		"healthmonitor": heathmonitor,
	}
	_, err = self.lb.region.put(SERVICE_ELB, fmt.Sprintf("elb/healthmonitors/%s", lbbg.HealthMonitorID), params)
	return err
}

func (self *SElbListener) Delete(ctx context.Context) error {
	return self.lb.region.DeleteElbListener(self.ID)
}

func (self *SRegion) DeleteElbListener(id string) error {
	_, err := self.delete(SERVICE_ELB, "elb/listeners/"+id)
	return err
}

func (self *SRegion) UpdateLoadBalancerListener(listenerId string, listener *cloudprovider.SLoadbalancerListenerCreateOptions) error {
	params := map[string]interface{}{
		"name":            listener.Name,
		"description":     listener.Description,
		"http2_enable":    listener.EnableHTTP2,
		"default_pool_id": jsonutils.JSONNull,
	}
	if len(listener.BackendGroupId) > 0 {
		params["default_pool_id"] = listener.BackendGroupId
	}

	if listener.ListenerType == api.LB_LISTENER_TYPE_HTTPS {
		params["default_tls_container_ref"] = listener.CertificateId
	}

	if listener.XForwardedFor {
		params["insert_headers"] = map[string]interface{}{
			"X-Forwarded-ELB-IP": listener.XForwardedFor,
		}
	}
	_, err := self.put(SERVICE_ELB, "elb/listeners/"+listenerId, map[string]interface{}{"listener": params})
	return err
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0136295315.html
func (self *SRegion) GetLoadBalancerPolicies(listenerId string) ([]SElbListenerPolicy, error) {
	query := url.Values{}
	if len(listenerId) > 0 {
		query.Set("listener_id", listenerId)
	}

	resp, err := self.list(SERVICE_ELB, "elb/l7policies", query)
	if err != nil {
		return nil, err
	}

	ret := []SElbListenerPolicy{}
	return ret, resp.Unmarshal(&ret, "l7policies")
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0116649234.html
func (self *SRegion) GetLoadBalancerPolicyRules(policyId string) ([]SElbListenerPolicyRule, error) {
	resp, err := self.list(SERVICE_ELB, fmt.Sprintf("elb/l7policies/%s/rules", policyId), url.Values{})
	if err != nil {
		return nil, err
	}

	ret := []SElbListenerPolicyRule{}
	return ret, resp.Unmarshal(&ret, "rules")
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0136295317.html
func (self *SRegion) CreateLoadBalancerPolicy(listenerID string, rule *cloudprovider.SLoadbalancerListenerRule) (*SElbListenerPolicy, error) {
	ret := &SElbListenerPolicy{}
	params := map[string]interface{}{
		"name":             rule.Name,
		"listener_id":      listenerID,
		"action":           "REDIRECT_TO_POOL",
		"redirect_pool_id": rule.BackendGroupId,
	}
	resp, err := self.post(SERVICE_ELB, "elb/l7policies", map[string]interface{}{"l7policy": params})
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(ret, "l7policy")
	if err != nil {
		return nil, err
	}
	if len(rule.Domain) > 0 {
		params := map[string]interface{}{
			"type":         "HOST_NAME",
			"value":        rule.Domain,
			"compare_type": "EQUAL_TO",
		}
		_, err := self.post(SERVICE_ELB, fmt.Sprintf("elb/l7policies/%s/rules", ret.GetId()), map[string]interface{}{"rule": params})
		if err != nil {
			return ret, err
		}
	}

	if len(rule.Path) > 0 {
		params := map[string]interface{}{
			"type":         "PATH",
			"value":        rule.Path,
			"compare_type": "EQUAL_TO",
		}
		_, err := self.post(SERVICE_ELB, fmt.Sprintf("elb/l7policies/%s/rules", ret.GetId()), map[string]interface{}{"rule": params})
		if err != nil {
			return ret, err
		}
	}

	return ret, nil
}

func (self *SElbListener) GetClientIdleTimeout() int {
	return 0
}

func (self *SElbListener) GetBackendConnectTimeout() int {
	return 0
}
