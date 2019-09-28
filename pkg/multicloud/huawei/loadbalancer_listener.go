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
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type InsertHeaders struct {
	XForwardedELBIP bool `json:"X-Forwarded-ELB-IP"`
}

type Loadbalancer struct {
	ID string `json:"id"`
}

type SElbListener struct {
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

func (self *SElbListener) GetMetadata() *jsonutils.JSONDict {
	return nil
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
	return &l7policy, nil
}

func (self *SElbListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	ret := &SElbListenerPolicy{}
	err := DoGet(self.lb.region.ecsClient.ElbL7policies.Get, ruleId, nil, ret)
	if err != nil {
		return nil, err
	}

	ret.region = self.lb.region
	ret.lb = self.lb
	ret.listener = self
	return ret, nil
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
		iret = append(iret, &ret[i])
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

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561544.html
/*
default_pool_id有如下限制：
不能更新为其他监听器的default_pool。
不能更新为其他监听器的关联的转发策略所使用的pool。
default_pool_id对应的后端云服务器组的protocol和监听器的protocol有如下关系：
监听器的protocol为TCP时，后端云服务器组的protocol必须为TCP。
监听器的protocol为UDP时，后端云服务器组的protocol必须为UDP。
监听器的protocol为HTTP或TERMINATED_HTTPS时，后端云服务器组的protocol必须为HTTP。
*/
func (self *SElbListener) Sync(listener *cloudprovider.SLoadbalancerListener) error {
	return self.lb.region.UpdateLoadBalancerListener(self.GetId(), listener)
}

func (self *SElbListener) Delete() error {
	err := DoDelete(self.lb.region.ecsClient.ElbListeners.Delete, self.GetId(), nil, nil)
	if err != nil {
		return err
	}

	return nil
}

func (self *SRegion) UpdateLoadBalancerListener(listenerId string, listener *cloudprovider.SLoadbalancerListener) error {
	params := jsonutils.NewDict()
	listenerObj := jsonutils.NewDict()
	listenerObj.Set("name", jsonutils.NewString(listener.Name))
	listenerObj.Set("description", jsonutils.NewString(listener.Description))
	listenerObj.Set("http2_enable", jsonutils.NewBool(listener.EnableHTTP2))
	if len(listener.BackendGroupID) > 0 {
		listenerObj.Set("default_pool_id", jsonutils.NewString(listener.BackendGroupID))
	} else {
		listenerObj.Set("default_pool_id", jsonutils.JSONNull)
	}

	if listener.ListenerType == api.LB_LISTENER_TYPE_HTTPS {
		listenerObj.Set("default_tls_container_ref", jsonutils.NewString(listener.CertificateID))
	}

	if listener.XForwardedFor {
		insertObj := jsonutils.NewDict()
		insertObj.Set("X-Forwarded-ELB-IP", jsonutils.NewBool(listener.XForwardedFor))
		listenerObj.Set("insert_headers", insertObj)
	}

	params.Set("listener", listenerObj)
	err := DoUpdate(self.ecsClient.ElbListeners.Update, listenerId, params, nil)
	if err != nil {
		return err
	}
	return nil
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0136295315.html
func (self *SRegion) GetLoadBalancerPolicies(listenerId string) ([]SElbListenerPolicy, error) {
	params := map[string]string{}
	if len(listenerId) > 0 {
		params["listener_id"] = listenerId
	}

	ret := []SElbListenerPolicy{}
	err := doListAll(self.ecsClient.ElbL7policies.List, params, &ret)
	if err != nil {
		return nil, err
	}

	for i := range ret {
		ret[i].region = self
	}

	return ret, nil
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0116649234.html
func (self *SRegion) GetLoadBalancerPolicyRules(policyId string) ([]SElbListenerPolicyRule, error) {
	m := self.ecsClient.ElbPolicies
	err := m.SetL7policyId(policyId)
	if err != nil {
		return nil, err
	}

	ret := []SElbListenerPolicyRule{}
	err = doListAll(m.List, nil, &ret)
	if err != nil {
		return nil, err
	}

	for i := range ret {
		ret[i].region = self
	}

	return ret, nil
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0136295317.html
func (self *SRegion) CreateLoadBalancerPolicy(listenerID string, rule *cloudprovider.SLoadbalancerListenerRule) (
	SElbListenerPolicy, error) {
	l7policy := SElbListenerPolicy{}
	params := jsonutils.NewDict()
	policyObj := jsonutils.NewDict()
	policyObj.Set("name", jsonutils.NewString(rule.Name))
	policyObj.Set("listener_id", jsonutils.NewString(listenerID))
	// todo: REDIRECT_TO_LISTENER?
	policyObj.Set("action", jsonutils.NewString("REDIRECT_TO_POOL"))
	policyObj.Set("redirect_pool_id", jsonutils.NewString(rule.BackendGroupID))
	params.Set("l7policy", policyObj)

	err := DoCreate(self.ecsClient.ElbL7policies.Create, params, &l7policy)
	if err != nil {
		return l7policy, err
	}

	m := self.ecsClient.ElbPolicies
	m.SetL7policyId(l7policy.GetId())
	if len(rule.Domain) > 0 {
		p := jsonutils.NewDict()
		p.Set("type", jsonutils.NewString("HOST_NAME"))
		p.Set("value", jsonutils.NewString(rule.Domain))
		// todo: support more compare_type
		p.Set("compare_type", jsonutils.NewString("EQUAL_TO"))
		rule := jsonutils.NewDict()
		rule.Set("rule", p)
		err := DoCreate(m.Create, rule, nil)
		if err != nil {
			return l7policy, err
		}
	}

	if len(rule.Path) > 0 {
		p := jsonutils.NewDict()
		p.Set("type", jsonutils.NewString("PATH"))
		p.Set("value", jsonutils.NewString(rule.Path))
		p.Set("compare_type", jsonutils.NewString("EQUAL_TO"))
		rule := jsonutils.NewDict()
		rule.Set("rule", p)
		err := DoCreate(m.Create, rule, nil)
		if err != nil {
			return l7policy, err
		}
	}

	return l7policy, nil
}
