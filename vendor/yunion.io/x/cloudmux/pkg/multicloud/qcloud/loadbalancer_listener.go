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

package qcloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var HTTP_CODES = []string{
	api.LB_HEALTH_CHECK_HTTP_CODE_1xx,
	api.LB_HEALTH_CHECK_HTTP_CODE_2xx,
	api.LB_HEALTH_CHECK_HTTP_CODE_3xx,
	api.LB_HEALTH_CHECK_HTTP_CODE_4xx,
	api.LB_HEALTH_CHECK_HTTP_CODE_5xx,
}

const (
	PROTOCOL_TCP     = "TCP"
	PROTOCOL_UDP     = "UDP"
	PROTOCOL_TCP_SSL = "TCP_SSL"
	PROTOCOL_HTTP    = "HTTP"
	PROTOCOL_HTTPS   = "HTTPS"
)

type certificate struct {
	SSLMode  string `json:"SSLMode"`
	CERTCAId string `json:"CertCaId"`
	CERTId   string `json:"CertId"`
}

/*
健康检查状态码（仅适用于HTTP/HTTPS转发规则）。可选值：1~31，默认 31。
1 表示探测后返回值 1xx 表示健康，2 表示返回 2xx 表示健康，4 表示返回 3xx 表示健康，8 表示返回 4xx 表示健康，16 表示返回 5xx 表示健康。
若希望多种码都表示健康，则将相应的值相加。
*/
type healthCheck struct {
	HTTPCheckDomain string `json:"HttpCheckDomain"`
	HealthSwitch    int    `json:"HealthSwitch"`
	HTTPCheckPath   string `json:"HttpCheckPath"`
	HTTPCheckMethod string `json:"HttpCheckMethod"`
	UnHealthNum     int    `json:"UnHealthNum"`
	IntervalTime    int    `json:"IntervalTime"`
	HTTPCode        int    `json:"HttpCode"` // 健康检查状态码（仅适用于HTTP/HTTPS转发规则）。可选值：1~31，默认 31。
	HealthNum       int    `json:"HealthNum"`
	TimeOut         int    `json:"TimeOut"`
}

type SLBListener struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	QcloudTags
	lb *SLoadbalancer

	Protocol          string            `json:"Protocol"` // 监听器协议类型，取值 TCP | UDP | HTTP | HTTPS | TCP_SSL
	Certificate       certificate       `json:"Certificate"`
	SniSwitch         int64             `json:"SniSwitch"`   // 是否开启SNI特性（本参数仅对于HTTPS监听器有意义）
	HealthCheck       healthCheck       `json:"HealthCheck"` // 仅适用于TCP/UDP/TCP_SSL监听器
	ListenerId        string            `json:"ListenerId"`
	ListenerName      string            `json:"ListenerName"`
	Rules             []SLBListenerRule `json:"Rules"` // 监听器下的全部转发规则（本参数仅对于HTTP/HTTPS监听器有意义）
	Scheduler         string            `json:"Scheduler"`
	SessionExpireTime int               `json:"SessionExpireTime"` // 会话保持时间，单位：秒。可选值：30~3600，默认 0，表示不开启。此参数仅适用于TCP/UDP监听器。
	Port              int               `json:"Port"`
}

// 腾讯云后端端口不是与listener绑定的
func (self *SLBListener) GetBackendServerPort() int {
	return 0
}

// https://cloud.tencent.com/document/product/214/30691
func (self *SLBListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	hc := getListenerRuleHealthCheck(rule)
	requestId, err := self.lb.region.CreateLoadbalancerListenerRule(self.lb.GetId(),
		self.GetId(),
		rule.Domain,
		rule.Path,
		rule.Scheduler,
		rule.StickySessionCookieTimeout,
		hc)
	if err != nil {
		return nil, err
	}

	err = self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
	if err != nil {
		return nil, err
	}

	err = self.Refresh()
	if err != nil {
		return nil, err
	}

	for _, r := range self.Rules {
		if r.GetPath() == rule.Path {
			r.listener = self
			return &r, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SLBListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := self.GetILoadbalancerListenerRules()
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if rule.GetId() == ruleId {
			return rule, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SLBListener) Start() error {
	return nil
}

func (self *SLBListener) Stop() error {
	return cloudprovider.ErrNotSupported
}

// https://cloud.tencent.com/document/product/214/30677
func (self *SLBListener) Sync(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) error {
	hc := getHealthCheck(listener)
	cert := getCertificate(listener)
	requestId, err := self.lb.region.UpdateLoadbalancerListener(
		self.lb.Forward,
		self.lb.GetId(),
		self.GetId(),
		&listener.Name,
		getScheduler(listener),
		&listener.StickySessionCookieTimeout,
		hc,
		cert)
	if err != nil {
		return err
	}

	return self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
}

func (self *SLBListener) Delete(ctx context.Context) error {
	requestId, err := self.lb.region.DeleteLoadbalancerListener(self.lb.Forward, self.lb.GetId(), self.GetId())
	if err != nil {
		return err
	}

	return self.lb.region.WaitLBTaskSuccess(requestId, 5*time.Second, 60*time.Second)
}

func (self *SLBListener) GetId() string {
	return self.ListenerId
}

func (self *SLBListener) GetName() string {
	return self.ListenerName
}

func (self *SLBListener) GetGlobalId() string {
	return self.ListenerId
}

// 腾讯云负载均衡没有启用禁用操作
func (self *SLBListener) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLBListener) Refresh() error {
	listener, err := self.lb.region.GetLoadbalancerListener(self.lb.LoadBalancerId, self.ListenerId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, listener)
}

func (self *SLBListener) GetEgressMbps() int {
	return 0
}

func (self *SLBListener) GetListenerType() string {
	switch self.Protocol {
	case PROTOCOL_TCP:
		return api.LB_LISTENER_TYPE_TCP
	case PROTOCOL_UDP:
		return api.LB_LISTENER_TYPE_UDP
	case PROTOCOL_HTTP:
		return api.LB_LISTENER_TYPE_HTTP
	case PROTOCOL_HTTPS:
		return api.LB_LISTENER_TYPE_HTTPS
	case PROTOCOL_TCP_SSL:
		return api.LB_LISTENER_TYPE_TCP
	default:
		return ""
	}
}

func (self *SLBListener) GetListenerPort() int {
	return self.Port
}

func (self *SLBListener) GetScheduler() string {
	switch strings.ToLower(self.Scheduler) {
	case "wrr":
		return api.LB_SCHEDULER_WRR
	case "ip_hash":
		return api.LB_SCHEDULER_SCH
	case "least_conn":
		return api.LB_SCHEDULER_WLC
	default:
		return ""
	}
}

func (self *SLBListener) GetAclStatus() string {
	return api.LB_BOOL_OFF
}

func (self *SLBListener) GetAclType() string {
	return ""
}

func (self *SLBListener) GetAclId() string {
	return ""
}

func (self *SLBListener) GetHealthCheck() string {
	if self.HealthCheck.HealthSwitch == 0 {
		return api.LB_BOOL_OFF
	}
	return api.LB_BOOL_ON
}

func (self *SLBListener) GetHealthCheckType() string {
	if len(self.HealthCheck.HTTPCheckMethod) > 0 {
		return api.LB_HEALTH_CHECK_HTTP
	}
	return api.LB_HEALTH_CHECK_TCP
}

func (self *SLBListener) GetHealthCheckTimeout() int {
	return self.HealthCheck.TimeOut
}

func (self *SLBListener) GetHealthCheckInterval() int {
	return self.HealthCheck.IntervalTime
}

func (self *SLBListener) GetHealthCheckRise() int {
	return self.HealthCheck.HealthNum
}

func (self *SLBListener) GetHealthCheckFail() int {
	return self.HealthCheck.UnHealthNum
}

func (self *SLBListener) GetHealthCheckReq() string {
	return ""
}

func (self *SLBListener) GetHealthCheckExp() string {
	return ""
}

func (self *SLBListener) GetBackendGroup() *SLBBackendGroup {
	return &SLBBackendGroup{lb: self.lb, listener: self}
}

func (self *SLBListener) GetBackendGroupId() string {
	bg := self.GetBackendGroup()
	return bg.GetGlobalId()
}

func (self *SLBListener) GetHealthCheckDomain() string {
	return self.HealthCheck.HTTPCheckDomain
}

func (self *SLBListener) GetHealthCheckURI() string {
	return self.HealthCheck.HTTPCheckPath
}

func (self *SLBListener) GetHealthCheckCode() string {
	codes := []string{}
	for i := uint8(0); i < 5; i++ {
		n := 1 << i
		if (self.HealthCheck.HTTPCode & n) == n {
			codes = append(codes, HTTP_CODES[i])
		}
	}

	return strings.Join(codes, ",")
}

// 仅http、https类型监听包含rules
func (self *SLBListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules := self.Rules
	iRules := []cloudprovider.ICloudLoadbalancerListenerRule{}
	for i := 0; i < len(rules); i++ {
		rules[i].listener = self
		iRules = append(iRules, &rules[i])
	}
	return iRules, nil
}

func (self *SLBListener) GetStickySession() string {
	if self.SessionExpireTime == 0 {
		return api.LB_BOOL_OFF
	}
	return api.LB_BOOL_ON
}

// 支持基于 cookie 插入的会话保持能力 https://cloud.tencent.com/document/product/214/6154
func (self *SLBListener) GetStickySessionType() string {
	return api.LB_STICKY_SESSION_TYPE_INSERT
}

// https://cloud.tencent.com/document/product/214/2736
// 经测试应用型负载均衡返回都是 tgw_l7_route。
func (self *SLBListener) GetStickySessionCookie() string {
	if self.GetListenerType() == api.LB_LISTENER_TYPE_HTTPS {
		return "tgw_l7_route"
	}

	return ""
}

func (self *SLBListener) GetStickySessionCookieTimeout() int {
	return self.SessionExpireTime
}

/*
7层负载均衡系统提供 X-Forwarded-For 的方式获取访问者真实 IP，LB 侧默认开启

https://cloud.tencent.com/document/product/214/6151
七层转发获取来访真实IP的方法 https://cloud.tencent.com/document/product/214/3728
*/
func (self *SLBListener) XForwardedForEnabled() bool {
	switch self.GetListenerType() {
	case api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS:
		return true
	default:
		return false
	}
}

// HTTP/HTTPS协议默认支持用户开启gzip压缩功能
// 负载均衡开启Gzip配置及检测方法说明 https://cloud.tencent.com/document/product/214/5404
func (self *SLBListener) GzipEnabled() bool {
	switch self.GetListenerType() {
	case api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS:
		return true
	default:
		return false
	}
}

func (self *SLBListener) GetCertificateId() string {
	return self.Certificate.CERTId
}

// https://cloud.tencent.com/document/product/214/5412#2.-https.E6.94.AF.E6.8C.81.E5.93.AA.E4.BA.9B.E7.89.88.E6.9C.AC.E7.9A.84ssl.2Ftls.E5.AE.89.E5.85.A8.E5.8D.8F.E8.AE.AE.EF.BC.9F
func (self *SLBListener) GetTLSCipherPolicy() string {
	return ""
}

// 负载均衡能力说明 https://cloud.tencent.com/document/product/214/6534
func (self *SLBListener) HTTP2Enabled() bool {
	return true
}

func (self *SRegion) GetLoadbalancerListener(lbId, lisId string) (*SLBListener, error) {
	ret, err := self.GetLoadbalancerListeners(lbId, []string{lisId}, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetLoadbalancerListeners")
	}
	for i := range ret {
		if ret[i].ListenerId == lisId {
			return &ret[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, lisId)
}

func (self *SRegion) GetLoadbalancerListeners(lbId string, lblisIds []string, protocol string) ([]SLBListener, error) {
	params := map[string]string{
		"LoadBalancerId": lbId,
	}
	if len(protocol) > 0 {
		params["Protocol"] = protocol
	}
	for i, id := range lblisIds {
		params[fmt.Sprintf("ListenerIds.%d", i)] = id
	}
	listeners := []SLBListener{}
	resp, err := self.clbRequest("DescribeListeners", params)
	if err != nil {
		return nil, err
	}

	err = resp.Unmarshal(&listeners, "Listeners")
	if err != nil {
		return nil, err
	}

	return listeners, nil
}

// 返回requestId
func (self *SRegion) CreateLoadbalancerListenerRule(lbid string, listenerId string, domain string, url string, scheduler string, sessionExpireTime int, hc *healthCheck) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("loadbalancer id should not be empty")
	}

	params := map[string]string{
		"LoadBalancerId": lbid,
		"ListenerId":     listenerId,
		"Rules.0.Domain": domain,
		"Rules.0.Url":    url,
	}

	params["Rules.0.Scheduler"] = scheduler
	params["Rules.0.SessionExpireTime"] = strconv.Itoa(sessionExpireTime)

	// health check
	params["Rules.0.HealthCheck.HealthSwitch"] = strconv.Itoa(hc.HealthSwitch)
	params["Rules.0.HealthCheck.IntervalTime"] = strconv.Itoa(hc.IntervalTime)
	params["Rules.0.HealthCheck.HealthNum"] = strconv.Itoa(hc.HealthNum)
	params["Rules.0.HealthCheck.UnHealthNum"] = strconv.Itoa(hc.UnHealthNum)
	if hc.HTTPCode > 0 {
		params["Rules.0.HealthCheck.HttpCode"] = strconv.Itoa(hc.HTTPCode)
		params["Rules.0.HealthCheck.HttpCheckPath"] = hc.HTTPCheckPath
		params["Rules.0.HealthCheck.HttpCheckDomain"] = hc.HTTPCheckDomain
		params["Rules.0.HealthCheck.HttpCheckMethod"] = hc.HTTPCheckMethod
	}

	resp, err := self.clbRequest("CreateRule", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

// 返回requestId
func (self *SRegion) deleteLoadbalancerListener(lbid string, listenerId string) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("loadbalancer id should not be empty")
	}

	params := map[string]string{
		"LoadBalancerId": lbid,
		"ListenerId":     listenerId,
	}

	resp, err := self.clbRequest("DeleteListener", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

// 返回requestId
func (self *SRegion) DeleteLoadbalancerListener(t LB_TYPE, lbid string, listenerId string) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("loadbalancer id should not be empty")
	}

	return self.deleteLoadbalancerListener(lbid, listenerId)
}

// https://cloud.tencent.com/document/product/214/30681
func (self *SRegion) updateLoadbalancerListener(lbid string, listenerId string, listenerName *string, scheduler *string, sessionExpireTime *int, healthCheck *healthCheck, cert *certificate) (string, error) {
	params := map[string]string{
		"LoadBalancerId": lbid,
		"ListenerId":     listenerId,
	}

	if listenerName != nil && len(*listenerName) > 0 {
		params["ListenerName"] = *listenerName
	}

	if scheduler != nil && len(*scheduler) > 0 {
		params["Scheduler"] = *scheduler
	}

	if sessionExpireTime != nil {
		params["SessionExpireTime"] = strconv.Itoa(*sessionExpireTime)
	}

	params = healthCheckParams(LB_TYPE_APPLICATION, params, healthCheck, "HealthCheck.")

	resp, err := self.clbRequest("ModifyListener", params)
	if err != nil {
		return "", err
	}

	return resp.GetString("RequestId")
}

func (self *SRegion) UpdateLoadbalancerListener(t LB_TYPE, lbid string, listenerId string, listenerName *string, scheduler *string, sessionExpireTime *int, healthCheck *healthCheck, cert *certificate) (string, error) {
	if len(lbid) == 0 {
		return "", fmt.Errorf("loadbalancer id should not be empty")
	}

	if len(listenerId) == 0 {
		return "", fmt.Errorf("loadbalancer listener id should not be empty")
	}

	return self.updateLoadbalancerListener(lbid, listenerId, listenerName, scheduler, sessionExpireTime, healthCheck, cert)
}

func getHealthCheck(listener *cloudprovider.SLoadbalancerListenerCreateOptions) *healthCheck {
	var hc *healthCheck
	if listener.HealthCheck == api.LB_BOOL_ON {
		hc = &healthCheck{
			HealthSwitch: 1,
			UnHealthNum:  listener.HealthCheckFail,
			IntervalTime: listener.HealthCheckInterval,
			HealthNum:    listener.HealthCheckRise,
			TimeOut:      listener.HealthCheckTimeout,
		}

		httpCode := onecloudHealthCodeToQcloud(listener.HealthCheckHttpCode)
		if httpCode > 0 {
			hc.HTTPCode = httpCode
			hc.HTTPCheckMethod = "HEAD" // todo: add column HttpCheckMethod in model
			hc.HTTPCheckDomain = listener.HealthCheckDomain
			hc.HTTPCheckPath = listener.HealthCheckURI
		}
	} else {
		hc = &healthCheck{
			HealthSwitch: 0,
			UnHealthNum:  3,
			IntervalTime: 5,
			HealthNum:    3,
			TimeOut:      2,
		}
	}

	return hc
}

func getListenerRuleHealthCheck(rule *cloudprovider.SLoadbalancerListenerRule) *healthCheck {
	var hc *healthCheck
	if rule.HealthCheck == api.LB_BOOL_ON {
		hc = &healthCheck{
			HealthSwitch: 1,
			UnHealthNum:  rule.HealthCheckFail,
			IntervalTime: rule.HealthCheckInterval,
			HealthNum:    rule.HealthCheckRise,
			TimeOut:      rule.HealthCheckTimeout,
		}

		httpCode := onecloudHealthCodeToQcloud(rule.HealthCheckHttpCode)
		if httpCode > 0 {
			hc.HTTPCode = httpCode
			hc.HTTPCheckMethod = "HEAD" // todo: add column HttpCheckMethod in model
			hc.HTTPCheckDomain = rule.HealthCheckDomain
			hc.HTTPCheckPath = rule.HealthCheckURI
		}
	} else {
		hc = &healthCheck{
			HealthSwitch: 0,
			UnHealthNum:  3,
			IntervalTime: 5,
			HealthNum:    3,
			TimeOut:      2,
		}
	}

	return hc
}

func getCertificate(listener *cloudprovider.SLoadbalancerListenerCreateOptions) *certificate {
	var cert *certificate
	if len(listener.CertificateId) > 0 {
		cert = &certificate{
			SSLMode:  "UNIdIRECTIONAL",
			CERTCAId: "",
			CERTId:   listener.CertificateId,
		}
	}

	return cert
}

func getProtocol(listener *cloudprovider.SLoadbalancerListenerCreateOptions) string {
	switch listener.ListenerType {
	case api.LB_LISTENER_TYPE_HTTPS:
		return PROTOCOL_HTTPS
	case api.LB_LISTENER_TYPE_HTTP:
		return PROTOCOL_HTTP
	case api.LB_LISTENER_TYPE_TCP:
		return PROTOCOL_TCP
	case api.LB_LISTENER_TYPE_UDP:
		return PROTOCOL_UDP
	case "tcp_ssl":
		return PROTOCOL_TCP_SSL
	default:
		return ""
	}
}

func getClassicLBProtocol(listener *cloudprovider.SLoadbalancerListenerCreateOptions) int {
	switch listener.ListenerType {
	case api.LB_LISTENER_TYPE_HTTP:
		return 1
	case api.LB_LISTENER_TYPE_HTTPS:
		return 4
	case api.LB_LISTENER_TYPE_TCP:
		return 2
	case api.LB_LISTENER_TYPE_UDP:
		return 3
	default:
		return 0 // 非法值
	}
}

func getScheduler(listener *cloudprovider.SLoadbalancerListenerCreateOptions) *string {
	var sch string
	switch listener.Scheduler {
	case api.LB_SCHEDULER_WRR:
		sch = "WRR"
	case api.LB_SCHEDULER_WLC:
		sch = "LEAST_CONN"
	case api.LB_SCHEDULER_SCH:
		sch = "IP_HASH"
	default:
		return nil
	}

	return &sch
}

func healthCheckParams(t LB_TYPE, params map[string]string, hc *healthCheck, paramPrefix string) map[string]string {
	if hc == nil {
		return params
	}

	if t == LB_TYPE_APPLICATION {
		params[paramPrefix+"HealthSwitch"] = strconv.Itoa(hc.HealthSwitch)
		params[paramPrefix+"TimeOut"] = strconv.Itoa(hc.TimeOut)
		params[paramPrefix+"IntervalTime"] = strconv.Itoa(hc.IntervalTime)
		params[paramPrefix+"HealthNum"] = strconv.Itoa(hc.HealthNum)
		params[paramPrefix+"UnHealthNum"] = strconv.Itoa(hc.UnHealthNum)
		if hc.HTTPCode > 0 {
			params[paramPrefix+"HttpCode"] = strconv.Itoa(hc.HTTPCode)
			params[paramPrefix+"HttpCheckPath"] = hc.HTTPCheckPath
			params[paramPrefix+"HttpCheckDomain"] = hc.HTTPCheckDomain
			params[paramPrefix+"HttpCheckMethod"] = hc.HTTPCheckMethod
		}
	} else {
		params[paramPrefix+"healthSwitch"] = strconv.Itoa(hc.HealthSwitch)
		params[paramPrefix+"timeOut"] = strconv.Itoa(hc.TimeOut)
		params[paramPrefix+"intervalTime"] = strconv.Itoa(hc.IntervalTime)
		params[paramPrefix+"healthNum"] = strconv.Itoa(hc.HealthNum)
		params[paramPrefix+"unHealthNum"] = strconv.Itoa(hc.UnHealthNum)
		if hc.HTTPCode > 0 {
			params[paramPrefix+"httpCode"] = strconv.Itoa(hc.HTTPCode)
			params[paramPrefix+"httpCheckPath"] = hc.HTTPCheckPath
		}
	}
	return params
}

func (self *SLBListener) GetClientIdleTimeout() int {
	return 0
}

func (self *SLBListener) GetBackendConnectTimeout() int {
	return 0
}
