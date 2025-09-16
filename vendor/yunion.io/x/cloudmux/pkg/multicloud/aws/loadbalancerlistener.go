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

package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElbListeners struct {
	NextMarker string
	Listeners  []SElbListener `xml:"Listeners>member"`
}

type SElbListener struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	AwsTags
	lb    *SElb
	group *SElbBackendGroup

	Port            int             `json:"Port"`
	Protocol        string          `json:"Protocol"`
	DefaultActions  []DefaultAction `json:"DefaultActions"`
	SSLPolicy       string          `json:"SslPolicy"`
	Certificates    []Certificate   `json:"Certificates"`
	LoadBalancerArn string          `json:"LoadBalancerArn"`
	ListenerArn     string          `json:"ListenerArn"`
}

type Certificate struct {
	CertificateArn string `json:"CertificateArn"`
}

type DefaultAction struct {
	TargetGroupArn string `json:"TargetGroupArn"`
	Type           string `json:"Type"`
}

func (self *SElbListener) GetId() string {
	return self.ListenerArn
}

func (self *SElbListener) GetName() string {
	segs := strings.Split(self.ListenerArn, "/")
	return segs[len(segs)-1]
}

func (self *SElbListener) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbListener) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbListener) Refresh() error {
	listener, err := self.lb.region.GetElbListener(self.GetId())
	if err != nil {
		return err
	}

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
	return ""
}

func (self *SElbListener) GetListenerType() string {
	switch self.Protocol {
	case "TCP":
		return api.LB_LISTENER_TYPE_TCP
	case "UDP":
		return api.LB_LISTENER_TYPE_UDP
	case "HTTP":
		return api.LB_LISTENER_TYPE_HTTP
	case "HTTPS":
		return api.LB_LISTENER_TYPE_HTTPS
	case "TCP_SSL":
		return api.LB_LISTENER_TYPE_TCP
	case "TCP_UDP":
		return api.LB_LISTENER_TYPE_TCP_UDP
	default:
		return ""
	}
}

func (self *SElbListener) GetListenerPort() int {
	return self.Port
}

func (self *SElbListener) GetScheduler() string {
	// api.LB_SCHEDULER_RR ?
	return ""
}

func (self *SElbListener) GetAclStatus() string {
	return api.LB_BOOL_OFF
}

func (self *SElbListener) GetAclType() string {
	return ""
}

func (self *SElbListener) GetAclId() string {
	return ""
}

func (self *SElbListener) GetEgressMbps() int {
	return 0
}

func (self *SElbListener) GetHealthCheck() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return ""
	}

	return health.HealthCheck
}

func (self *SElbListener) getBackendGroup() (*SElbBackendGroup, error) {
	if self.group != nil {
		return self.group, nil
	}

	lbbg, err := self.lb.region.GetElbBackendgroup(self.DefaultActions[0].TargetGroupArn)
	if err != nil {
		return nil, errors.Wrap(err, "GetElbBackendgroup")
	}

	self.group = lbbg
	return self.group, nil
}

func (self *SElbListener) GetHealthCheckType() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return ""
	}

	return health.HealthCheckType
}

func (self *SElbListener) GetHealthCheckTimeout() int {
	group, err := self.getBackendGroup()
	if err != nil {
		return 0
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return 0
	}

	return health.HealthCheckTimeout
}

func (self *SElbListener) GetHealthCheckInterval() int {
	group, err := self.getBackendGroup()
	if err != nil {
		return 0
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return 0
	}

	return health.HealthCheckInterval
}

func (self *SElbListener) GetHealthCheckRise() int {
	group, err := self.getBackendGroup()
	if err != nil {
		return 0
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return 0
	}

	return health.HealthCheckRise
}

func (self *SElbListener) GetHealthCheckFail() int {
	group, err := self.getBackendGroup()
	if err != nil {
		return 0
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return 0
	}

	return health.HealthCheckFail
}

func (self *SElbListener) GetHealthCheckReq() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return ""
	}

	return health.HealthCheckReq
}

func (self *SElbListener) GetHealthCheckExp() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return ""
	}

	return health.HealthCheckExp
}

func (self *SElbListener) GetBackendGroupId() string {
	return self.DefaultActions[0].TargetGroupArn
}

func (self *SElbListener) GetBackendServerPort() int {
	return 0
}

func (self *SElbListener) GetHealthCheckDomain() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return ""
	}

	return health.HealthCheckDomain
}

func (self *SElbListener) GetHealthCheckURI() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return ""
	}

	return health.HealthCheckURI
}

func (self *SElbListener) GetHealthCheckCode() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	health, err := group.GetHealthCheck()
	if err != nil {
		return ""
	}

	return health.HealthCheckHttpCode
}

func (self *SElbListener) ChangeCertificate(ctx context.Context, opts *cloudprovider.ListenerCertificateOptions) error {
	params := map[string]string{
		"ListenerArn":                          self.ListenerArn,
		"Certificates.member.1.CertificateArn": opts.CertificateId,
	}
	var err error
	for i := 0; i < 3; i++ {
		err = self.lb.region.elbRequest("ModifyListener", params, nil)
		if err == nil {
			return nil
		}
		if strings.Contains(err.Error(), "CertificateNotFound") {
			time.Sleep(time.Second * 10)
			continue
		}
		return err
	}
	return err
}

func (listerner *SElbListener) SetAcl(ctx context.Context, opts *cloudprovider.ListenerAclOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElbListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := self.GetILoadbalancerListenerRules()
	if err != nil {
		return nil, errors.Wrap(err, "GetILoadbalancerListenerRules")
	} else {
		if err := self.lb.region.UpdateRulesPriority(rules); err != nil {
			return nil, errors.Wrap(err, "UpdateRulesPriority")
		}
	}

	ret, err := self.lb.region.CreateElbListenerRule(self.GetId(), rule)
	if err != nil {
		return nil, errors.Wrap(err, "CreateElbListenerRule")
	}

	ret.listener = self
	return ret, nil
}

func (self *SElbListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rule, err := self.lb.region.GetElbListenerRule(ruleId)
	if err != nil {
		return nil, errors.Wrap(err, "GetElbListenerRule")
	}
	rule.listener = self
	return rule, nil
}

func (self *SElbListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	ret := []cloudprovider.ICloudLoadbalancerListenerRule{}
	marker := ""
	for {
		part, marker, err := self.lb.region.GetElbListenerRules(self.ListenerArn, "", marker)
		if err != nil {
			return nil, err
		}
		for i := range part {
			part[i].listener = self
			ret = append(ret, &part[i])
		}
		if len(marker) == 0 || len(part) == 0 {
			break
		}
	}
	return ret, nil
}

func (self *SElbListener) GetStickySession() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	session, err := group.GetStickySession()
	if err != nil {
		return ""
	}

	return session.StickySession
}

func (self *SElbListener) GetStickySessionType() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	session, err := group.GetStickySession()
	if err != nil {
		return ""
	}

	return session.StickySessionType
}

func (self *SElbListener) GetStickySessionCookie() string {
	group, err := self.getBackendGroup()
	if err != nil {
		return ""
	}

	session, err := group.GetStickySession()
	if err != nil {
		return ""
	}

	return session.StickySessionCookie
}

func (self *SElbListener) GetStickySessionCookieTimeout() int {
	group, err := self.getBackendGroup()
	if err != nil {
		return 0
	}

	session, err := group.GetStickySession()
	if err != nil {
		return 0
	}

	return session.StickySessionCookieTimeout
}

func (self *SElbListener) XForwardedForEnabled() bool {
	return false
}

func (self *SElbListener) GzipEnabled() bool {
	return false
}

func (self *SElbListener) GetCertificateId() string {
	if len(self.Certificates) > 0 {
		return self.Certificates[0].CertificateArn
	}

	return ""
}

func (self *SElbListener) GetTLSCipherPolicy() string {
	return self.SSLPolicy
}

func (self *SElbListener) HTTP2Enabled() bool {
	return false
}

func (self *SElbListener) GetClientIdleTimeout() int {
	return 0
}

func (self *SElbListener) GetBackendConnectTimeout() int {
	return 0
}

func (self *SElbListener) Start() error {
	return nil
}

func (self *SElbListener) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElbListener) ChangeScheduler(ctx context.Context, opts *cloudprovider.ChangeListenerSchedulerOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElbListener) SetHealthCheck(ctx context.Context, opts *cloudprovider.ListenerHealthCheckOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SElbListener) Delete(ctx context.Context) error {
	return self.lb.region.DeleteElbListener(self.GetId())
}

func (self *SRegion) GetElbListeners(elbId, lisId, marker string) ([]SElbListener, string, error) {
	ret := &SElbListeners{}
	params := map[string]string{}
	if len(elbId) > 0 {
		params["LoadBalancerArn"] = elbId
	}
	if len(lisId) > 0 {
		params["ListenerArns.member.1"] = lisId
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	err := self.elbRequest("DescribeListeners", params, ret)
	if err != nil {
		return nil, "", err
	}
	return ret.Listeners, ret.NextMarker, nil
}

func unmarshalAwsOutput(output interface{}, respKey string, result interface{}) error {
	_ret, err := json.Marshal(output)
	if err != nil {
		return err
	}

	obj, err := jsonutils.Parse(_ret)
	if err != nil {
		return err
	}

	if len(respKey) == 0 {
		err = obj.Unmarshal(result)
		if err != nil {
			return err
		}
	} else {
		err = obj.Unmarshal(result, respKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *SRegion) GetElbListener(listenerId string) (*SElbListener, error) {
	ret, _, err := self.GetElbListeners("", listenerId, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetElbListeners")
	}
	for i := range ret {
		if ret[i].ListenerArn == listenerId {
			return &ret[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, listenerId)
}

func (self *SRegion) CreateElbListener(lbId string, opts *cloudprovider.SLoadbalancerListenerCreateOptions) (*SElbListener, error) {
	params := map[string]string{
		"LoadBalancerArn":                        lbId,
		"Port":                                   fmt.Sprintf("%d", opts.ListenerPort),
		"Protocol":                               strings.ToUpper(opts.ListenerType),
		"DefaultActions.member.1.Type":           "forward",
		"DefaultActions.member.1.TargetGroupArn": opts.BackendGroupId,
	}
	if opts.ListenerType == api.LB_LISTENER_TYPE_HTTPS {
		params["Certificates.member.1.CertificateArn"] = opts.CertificateId
		params["SslPolicy"] = "ELBSecurityPolicy-2016-08"
	}
	ret := &SElbListeners{}
	for i := 0; i < 4; i++ {
		err := self.elbRequest("CreateListener", params, ret)
		if err == nil {
			break
		}
		// aws 比较诡异，证书能查询到，但是如果立即创建会报错，这里只能等待一会重试
		if strings.Contains(err.Error(), "CertificateNotFound") {
			time.Sleep(time.Second * 10)
			continue
		}
		return nil, errors.Wrapf(err, "CreateListener")
	}
	for i := range ret.Listeners {
		return &ret.Listeners[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (self *SRegion) GetElbListenerRules(listenerId string, ruleId, marker string) ([]SElbListenerRule, string, error) {
	params := map[string]string{}
	if len(listenerId) > 0 {
		params["ListenerArn"] = listenerId
	}
	if len(ruleId) > 0 {
		params["RuleArns.member.1"] = ruleId
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	ret := &SElbListenerRules{}
	err := self.elbRequest("DescribeRules", params, ret)
	if err != nil {
		return nil, "", errors.Wrapf(err, "DescribeRules")
	}
	return ret.Rules, ret.NextMarker, nil
}

func (self *SRegion) GetElbListenerRule(id string) (*SElbListenerRule, error) {
	rules, _, err := self.GetElbListenerRules("", id, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetElbListenerRules")
	}
	for i := range rules {
		if rules[i].RuleArn == id {
			return &rules[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) DeleteElbListener(id string) error {
	return self.elbRequest("DeleteListener", map[string]string{"ListenerArn": id}, nil)
}

func (self *SRegion) UpdateRulesPriority(rules []cloudprovider.ICloudLoadbalancerListenerRule) error {
	return nil
}

func (self *SElbListener) GetDescription() string {
	return self.AwsTags.GetDescription()
}
