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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SElbListener struct {
	region *SRegion
	lb     *SElb
	group  *SElbBackendGroup

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
	return self.ListenerArn
}

func (self *SElbListener) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbListener) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbListener) Refresh() error {
	listener, err := self.region.GetElbListener(self.GetId())
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

func (self *SElbListener) GetMetadata() *jsonutils.JSONDict {
	return jsonutils.NewDict()
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

	lbbg, err := self.region.GetElbBackendgroup(self.DefaultActions[0].TargetGroupArn)
	if err != nil {
		return nil, err
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

func (self *SElbListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := self.GetILoadbalancerListenerRules()
	if err != nil {
		return nil, err
	} else {
		if err := self.region.UpdateRulesPriority(rules); err != nil {
			return nil, err
		}
	}

	ret, err := self.region.CreateElbListenerRule(self.GetId(), rule)
	if err != nil {
		return nil, err
	}

	ret.listener = self
	ret.region = self.region
	return ret, nil
}

func (self *SElbListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rule, err := self.region.GetElbListenerRuleById(ruleId)
	if err != nil {
		return nil, err
	}

	rule.listener = self
	return rule, nil
}

func (self *SElbListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := self.region.GetElbListenerRules(self.GetId(), "")
	if err != nil {
		return nil, err
	}

	irules := make([]cloudprovider.ICloudLoadbalancerListenerRule, len(rules))
	for i := range rules {
		rules[i].listener = self
		irules[i] = &rules[i]
	}

	return irules, nil
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

func (self *SElbListener) Start() error {
	return nil
}

func (self *SElbListener) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElbListener) Sync(listener *cloudprovider.SLoadbalancerListener) error {
	return self.region.SyncElbListener(self, listener)
}

func (self *SElbListener) Delete() error {
	return self.region.DeleteElbListener(self.GetId())
}

func (self *SRegion) GetElbListeners(elbId string) ([]SElbListener, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	params := &elbv2.DescribeListenersInput{}
	params.SetLoadBalancerArn(elbId)
	ret, err := client.DescribeListeners(params)
	if err != nil {
		return nil, err
	}

	listeners := []SElbListener{}
	err = unmarshalAwsOutput(ret, "Listeners", &listeners)
	if err != nil {
		return nil, err
	}

	for i := range listeners {
		listeners[i].region = self
	}

	return listeners, nil
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
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	params := &elbv2.DescribeListenersInput{}
	params.SetListenerArns([]*string{&listenerId})
	ret, err := client.DescribeListeners(params)
	if err != nil {
		return nil, err
	}

	listeners := []SElbListener{}
	err = unmarshalAwsOutput(ret, "Listeners", &listeners)
	if err != nil {
		return nil, err
	}

	if len(listeners) == 1 {
		listeners[0].region = self
		return &listeners[0], nil
	}

	return nil, ErrorNotFound()
}

func (self *SRegion) CreateElbListener(listener *cloudprovider.SLoadbalancerListener) (*SElbListener, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	listenerType := strings.ToUpper(listener.ListenerType)
	params := &elbv2.CreateListenerInput{}
	params.SetLoadBalancerArn(listener.LoadbalancerID)
	params.SetPort(int64(listener.ListenerPort))
	params.SetProtocol(listenerType)
	action := &elbv2.Action{}
	action.SetType("forward")
	action.SetTargetGroupArn(listener.BackendGroupID)
	params.SetDefaultActions([]*elbv2.Action{action})
	if listenerType == "HTTPS" {
		cert := &elbv2.Certificate{
			CertificateArn: &listener.CertificateID,
		}

		params.SetCertificates([]*elbv2.Certificate{cert})
		params.SetSslPolicy("ELBSecurityPolicy-2016-08")
	}

	ret, err := client.CreateListener(params)
	if err != nil {
		// aws 比较诡异，证书能查询到，但是如果立即创建会报错，这里只能等待一会重试
		time.Sleep(10 * time.Second)
		if strings.Contains(err.Error(), "CertificateNotFound") {
			ret, err = client.CreateListener(params)
			if err != nil {
				return nil, errors.Wrap(err, "Region.CreateElbListener.Retry")
			}
		} else {
			return nil, errors.Wrap(err, "Region.CreateElbListener")
		}
	}

	listeners := []SElbListener{}
	err = unmarshalAwsOutput(ret, "Listeners", &listeners)
	if err != nil {
		return nil, err
	}

	if len(listeners) == 1 {
		listeners[0].region = self
		return &listeners[0], nil
	}

	return nil, fmt.Errorf("CreateElbListener err %#v", listeners)
}

func (self *SRegion) GetElbListenerRules(listenerId string, ruleId string) ([]SElbListenerRule, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	params := &elbv2.DescribeRulesInput{}
	if len(listenerId) > 0 {
		params.SetListenerArn(listenerId)
	}

	if len(ruleId) > 0 {
		params.SetRuleArns([]*string{&ruleId})
	}

	ret, err := client.DescribeRules(params)
	if err != nil {
		return nil, err
	}

	rules := []SElbListenerRule{}
	err = unmarshalAwsOutput(ret, "Rules", &rules)
	if err != nil {
		return nil, err
	}

	for i := range rules {
		rules[i].region = self
	}

	return rules, nil
}

func (self *SRegion) GetElbListenerRuleById(ruleId string) (*SElbListenerRule, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	params := &elbv2.DescribeRulesInput{}
	if len(ruleId) > 0 {
		params.SetRuleArns([]*string{&ruleId})
	}

	ret, err := client.DescribeRules(params)
	if err != nil {
		return nil, err
	}

	rules := []SElbListenerRule{}
	err = unmarshalAwsOutput(ret, "Rules", &rules)
	if err != nil {
		return nil, err
	}

	if len(rules) == 1 {
		rules[0].region = self
		return &rules[0], nil
	} else {
		log.Errorf("GetElbListenerRuleById %s %d found", ruleId, len(rules))
		return nil, ErrorNotFound()
	}
}

func (self *SRegion) DeleteElbListener(listenerId string) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return err
	}

	params := &elbv2.DeleteListenerInput{}
	params.SetListenerArn(listenerId)
	_, err = client.DeleteListener(params)
	if err != nil {
		return err
	}

	return nil
}

func (self *SRegion) SyncElbListener(listener *SElbListener, config *cloudprovider.SLoadbalancerListener) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return err
	}

	params := &elbv2.ModifyListenerInput{}
	params.SetListenerArn(listener.GetId())
	params.SetPort(int64(config.ListenerPort))
	params.SetProtocol(strings.ToUpper(config.ListenerType))
	action := &elbv2.Action{}
	action.SetType("forward")
	action.SetTargetGroupArn(config.BackendGroupID)
	params.SetDefaultActions([]*elbv2.Action{action})

	if config.ListenerType == api.LB_LISTENER_TYPE_HTTPS {
		cert := &elbv2.Certificate{}
		cert.SetCertificateArn(config.CertificateID)
		params.SetCertificates([]*elbv2.Certificate{cert})
	}

	_, err = client.ModifyListener(params)
	if err != nil {
		if strings.Contains(err.Error(), "CertificateNotFound") {
			// aws 比较诡异，证书能查询到，但是如果立即创建会报错，这里只能等待一会重试
			time.Sleep(10 * time.Second)
			_, err = client.ModifyListener(params)
			if err != nil {
				return errors.Wrap(err, "SRegion.SyncElbListener.ModifyListener.Retry")
			}
		}

		return errors.Wrap(err, "SRegion.SyncElbListener.ModifyListener")
	}

	hc := &cloudprovider.SLoadbalancerHealthCheck{
		HealthCheckType:     config.HealthCheckType,
		HealthCheckReq:      config.HealthCheckReq,
		HealthCheckExp:      config.HealthCheckExp,
		HealthCheck:         config.HealthCheck,
		HealthCheckTimeout:  config.HealthCheckTimeout,
		HealthCheckDomain:   config.HealthCheckDomain,
		HealthCheckHttpCode: config.HealthCheckHttpCode,
		HealthCheckURI:      config.HealthCheckURI,
		HealthCheckInterval: config.HealthCheckInterval,
		HealthCheckRise:     config.HealthCheckRise,
		HealthCheckFail:     config.HealthCheckFail,
	}
	err = self.modifyELbBackendGroup(config.BackendGroupID, hc)
	if err != nil {
		return errors.Wrap(err, "region.SyncElbListener.updateELbBackendGroup")
	}

	return nil
}

func (self *SRegion) UpdateRulesPriority(rules []cloudprovider.ICloudLoadbalancerListenerRule) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return err
	}

	ps := []*elbv2.RulePriorityPair{}
	for i := range rules {
		rule := rules[i].(*SElbListenerRule)
		if !rule.IsDefaultRule {
			v, _ := strconv.Atoi(rule.Priority)
			p := &elbv2.RulePriorityPair{}
			p.SetRuleArn(rules[i].GetId())
			p.SetPriority(int64(v + 1))

			ps = append(ps, p)
		}
	}

	if len(ps) == 0 {
		return nil
	}

	params := &elbv2.SetRulePrioritiesInput{}
	params.SetRulePriorities(ps)
	_, err = client.SetRulePriorities(params)
	if err != nil {
		return err
	}

	return nil
}
