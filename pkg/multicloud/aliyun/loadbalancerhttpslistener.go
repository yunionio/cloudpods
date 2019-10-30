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

package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerHTTPSListener struct {
	lb *SLoadbalancer

	ListenerPort      int    //	负载均衡实例前端使用的端口。
	BackendServerPort int    //	负载均衡实例后端使用的端口。
	Bandwidth         int    //	监听的带宽峰值。
	Status            string //	当前监听的状态。取值：starting | running | configuring | stopping | stopped
	Description       string

	XForwardedFor       string //	是否开启通过X-Forwarded-For头字段获取访者真实IP。
	XForwardedFor_SLBIP string //	是否通过SLB-IP头字段获取客户端请求的真实IP。
	XForwardedFor_SLBID string //	是否通过SLB-ID头字段获取负载均衡实例ID。
	XForwardedFor_proto string //	是否通过X-Forwarded-Proto头字段获取负载均衡实例的监听协议。
	Scheduler           string //	调度算法。
	StickySession       string //	是否开启会话保持。
	StickySessionType   string //	cookie的处理方式。
	CookieTimeout       int    //	Cookie超时时间。
	Cookie              string //	服务器上配置的cookie。
	AclStatus           string //	是否开启访问控制功能。取值：on | off（默认值）

	AclType string //	访问控制类型

	AclId string //	监听绑定的访问策略组ID。当AclStatus参数的值为on时，该参数必选。

	HealthCheck            string //	是否开启健康检查。
	HealthCheckDomain      string //	用于健康检查的域名。
	HealthCheckURI         string //	用于健康检查的URI。
	HealthyThreshold       int    //	健康检查阈值。
	UnhealthyThreshold     int    //	不健康检查阈值。
	HealthCheckTimeout     int    //	每次健康检查响应的最大超时间，单位为秒。
	HealthCheckInterval    int    //	健康检查的时间间隔，单位为秒。
	HealthCheckHttpCode    string //	健康检查正常的HTTP状态码。
	HealthCheckConnectPort int    //	健康检查的端口。
	VServerGroupId         string //	绑定的服务器组ID。
	ServerCertificateId    string //	服务器证书ID。
	CACertificateId        string //	CA证书ID。
	Gzip                   string //	是否开启Gzip压缩。
	Rules                  Rules  //监听下的转发规则列表，具体请参见RuleList。
	DomainExtensions       string //	域名扩展列表，具体请参见DomainExtensions。
	EnableHttp2            string //	是否开启HTTP/2特性。取值：on（默认值）|off

	TLSCipherPolicy string //
}

func (listener *SLoadbalancerHTTPSListener) GetName() string {
	if len(listener.Description) == 0 {
		listener.Refresh()
	}
	if len(listener.Description) > 0 {
		return listener.Description
	}
	return fmt.Sprintf("HTTPS:%d", listener.ListenerPort)
}

func (listerner *SLoadbalancerHTTPSListener) GetId() string {
	return fmt.Sprintf("%s/%d", listerner.lb.LoadBalancerId, listerner.ListenerPort)
}

func (listerner *SLoadbalancerHTTPSListener) GetGlobalId() string {
	return listerner.GetId()
}

func (listerner *SLoadbalancerHTTPSListener) GetStatus() string {
	switch listerner.Status {
	case "starting", "running":
		return api.LB_STATUS_ENABLED
	case "configuring", "stopping", "stopped":
		return api.LB_STATUS_DISABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (listerner *SLoadbalancerHTTPSListener) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (listerner *SLoadbalancerHTTPSListener) IsEmulated() bool {
	return false
}

func (listerner *SLoadbalancerHTTPSListener) GetEgressMbps() int {
	if listerner.Bandwidth < 1 {
		return 0
	}
	return listerner.Bandwidth
}

func (listerner *SLoadbalancerHTTPSListener) Refresh() error {
	lis, err := listerner.lb.region.GetLoadbalancerHTTPSListener(listerner.lb.LoadBalancerId, listerner.ListenerPort)
	if err != nil {
		return err
	}
	return jsonutils.Update(listerner, lis)
}

func (listerner *SLoadbalancerHTTPSListener) GetListenerType() string {
	return "https"
}

func (listerner *SLoadbalancerHTTPSListener) GetListenerPort() int {
	return listerner.ListenerPort
}

func (listerner *SLoadbalancerHTTPSListener) GetBackendGroupId() string {
	if len(listerner.VServerGroupId) == 0 {
		listerner.Refresh()
	}
	return listerner.VServerGroupId
}

func (listerner *SLoadbalancerHTTPSListener) GetBackendServerPort() int {
	return listerner.BackendServerPort
}

func (listerner *SLoadbalancerHTTPSListener) GetScheduler() string {
	return listerner.Scheduler
}

func (listerner *SLoadbalancerHTTPSListener) GetAclStatus() string {
	return listerner.AclStatus
}

func (listerner *SLoadbalancerHTTPSListener) GetAclType() string {
	return listerner.AclType
}

func (listerner *SLoadbalancerHTTPSListener) GetAclId() string {
	return listerner.AclId
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheck() string {
	return listerner.HealthCheck
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckType() string {
	return api.LB_HEALTH_CHECK_HTTP
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckDomain() string {
	return listerner.HealthCheckDomain
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckURI() string {
	return listerner.HealthCheckURI
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckCode() string {
	return listerner.HealthCheckHttpCode
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckRise() int {
	return listerner.HealthyThreshold
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckFail() int {
	return listerner.UnhealthyThreshold
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckTimeout() int {
	return listerner.HealthCheckTimeout
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckInterval() int {
	return listerner.HealthCheckInterval
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckReq() string {
	return ""
}

func (listerner *SLoadbalancerHTTPSListener) GetHealthCheckExp() string {
	return ""
}

func (listerner *SLoadbalancerHTTPSListener) GetStickySession() string {
	return listerner.StickySession
}

func (listerner *SLoadbalancerHTTPSListener) GetStickySessionType() string {
	return listerner.StickySessionType
}

func (listerner *SLoadbalancerHTTPSListener) GetStickySessionCookie() string {
	return listerner.Cookie
}

func (listerner *SLoadbalancerHTTPSListener) GetStickySessionCookieTimeout() int {
	return listerner.CookieTimeout
}

func (listerner *SLoadbalancerHTTPSListener) XForwardedForEnabled() bool {
	if listerner.XForwardedFor == "on" {
		return true
	}
	return false
}

func (listerner *SLoadbalancerHTTPSListener) GzipEnabled() bool {
	if listerner.Gzip == "on" {
		return true
	}
	return false
}

func (listerner *SLoadbalancerHTTPSListener) GetCertificateId() string {
	return listerner.ServerCertificateId
}

func (listerner *SLoadbalancerHTTPSListener) GetTLSCipherPolicy() string {
	return listerner.TLSCipherPolicy
}

func (listerner *SLoadbalancerHTTPSListener) HTTP2Enabled() bool {
	if listerner.EnableHttp2 == "on" {
		return true
	}
	return false
}

func (listerner *SLoadbalancerHTTPSListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := listerner.lb.region.GetLoadbalancerListenerRules(listerner.lb.LoadBalancerId, listerner.ListenerPort)
	if err != nil {
		return nil, err
	}
	iRules := []cloudprovider.ICloudLoadbalancerListenerRule{}
	for i := 0; i < len(rules); i++ {
		rules[i].httpsListener = listerner
		iRules = append(iRules, &rules[i])
	}
	return iRules, nil
}

func (region *SRegion) GetLoadbalancerHTTPSListener(loadbalancerId string, listenerPort int) (*SLoadbalancerHTTPSListener, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["ListenerPort"] = fmt.Sprintf("%d", listenerPort)
	body, err := region.lbRequest("DescribeLoadBalancerHTTPSListenerAttribute", params)
	if err != nil {
		return nil, err
	}
	listener := SLoadbalancerHTTPSListener{}
	return &listener, body.Unmarshal(&listener)
}

func (region *SRegion) constructHTTPCreateListenerParams(params map[string]string, listener *cloudprovider.SLoadbalancerListener) map[string]string {
	params["HealthCheck"] = listener.HealthCheck
	if listener.HealthCheck == "on" {
		if len(listener.HealthCheckURI) == 0 {
			params["HealthCheckURI"] = "/"
		}
		//The HealthCheckTimeout parameter is required.
		if listener.HealthCheckTimeout < 1 || listener.HealthCheckTimeout > 300 {
			listener.HealthCheckTimeout = 5
		}
		params["HealthCheckTimeout"] = fmt.Sprintf("%d", listener.HealthCheckTimeout)
	}
	params["RequestTimeout"] = fmt.Sprintf("%d", listener.ClientRequestTimeout)
	params["IdleTimeout"] = fmt.Sprintf("%d", listener.ClientIdleTimeout)

	params["StickySession"] = listener.StickySession
	params["StickySessionType"] = listener.StickySessionType
	params["Cookie"] = listener.StickySessionCookie
	if listener.StickySessionCookieTimeout < 1 || listener.StickySessionCookieTimeout > 86400 {
		listener.StickySessionCookieTimeout = 500
	}
	params["CookieTimeout"] = fmt.Sprintf("%d", listener.StickySessionCookieTimeout)
	//params["ForwardPort"] = fmt.Sprintf("%d", listener.ForwardPort) //暂不支持
	params["Gzip"] = "off"
	if listener.Gzip {
		params["Gzip"] = "on"
	}
	params["XForwardedFor"] = "off"
	if listener.XForwardedFor {
		params["XForwardedFor"] = "on"
	}
	return params
}

func (region *SRegion) CreateLoadbalancerHTTPSListener(lb *SLoadbalancer, listener *cloudprovider.SLoadbalancerListener) (cloudprovider.ICloudLoadbalancerListener, error) {
	params := region.constructBaseCreateListenerParams(lb, listener)
	params = region.constructHTTPCreateListenerParams(params, listener)
	params["ServerCertificateId"] = listener.CertificateID
	if listener.EnableHTTP2 {
		params["EnableHttp2"] = "on"
	} else {
		params["EnableHttp2"] = "off"
	}

	if len(listener.TLSCipherPolicy) > 0 {
		params["TLSCipherPolicy"] = listener.TLSCipherPolicy
	}
	_, err := region.lbRequest("CreateLoadBalancerHTTPSListener", params)
	if err != nil {
		return nil, err
	}
	iListener, err := region.GetLoadbalancerHTTPSListener(lb.LoadBalancerId, listener.ListenerPort)
	if err != nil {
		return nil, err
	}
	iListener.lb = lb
	return iListener, nil
}

func (listerner *SLoadbalancerHTTPSListener) Delete() error {
	return listerner.lb.region.DeleteLoadbalancerListener(listerner.lb.LoadBalancerId, listerner.ListenerPort)
}

func (listerner *SLoadbalancerHTTPSListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	_rule := &SLoadbalancerListenerRule{
		Domain:   rule.Domain,
		Url:      rule.Path,
		RuleName: rule.Name,
	}
	if len(rule.BackendGroupID) > 0 { //&& rule.BackendGroupType == api.LB_BACKENDGROUP_TYPE_NORMAL {
		_rule.VServerGroupId = rule.BackendGroupID
	}
	listenerRule, err := listerner.lb.region.CreateLoadbalancerListenerRule(listerner.ListenerPort, listerner.lb.LoadBalancerId, _rule)
	if err != nil {
		return nil, err
	}
	listenerRule.httpsListener = listerner
	return listenerRule, nil
}

func (listerner *SLoadbalancerHTTPSListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rule, err := listerner.lb.region.GetLoadbalancerListenerRule(ruleId)
	if err != nil {
		return nil, err
	}
	rule.httpsListener = listerner
	return rule, nil
}

func (listerner *SLoadbalancerHTTPSListener) Start() error {
	return listerner.lb.region.startListener(listerner.ListenerPort, listerner.lb.LoadBalancerId)
}

func (listerner *SLoadbalancerHTTPSListener) Stop() error {
	return listerner.lb.region.stopListener(listerner.ListenerPort, listerner.lb.LoadBalancerId)
}

func (region *SRegion) SyncLoadbalancerHTTPSListener(lb *SLoadbalancer, listener *cloudprovider.SLoadbalancerListener) error {
	params := region.constructBaseCreateListenerParams(lb, listener)
	params = region.constructHTTPCreateListenerParams(params, listener)
	params["ServerCertificateId"] = listener.CertificateID
	if listener.EnableHTTP2 {
		params["EnableHttp2"] = "on"
	} else {
		params["EnableHttp2"] = "off"
	}

	if len(lb.LoadBalancerSpec) > 0 && len(listener.TLSCipherPolicy) > 0 {
		params["TLSCipherPolicy"] = listener.TLSCipherPolicy
	}
	_, err := region.lbRequest("SetLoadBalancerHTTPSListenerAttribute", params)
	return err
}

func (listerner *SLoadbalancerHTTPSListener) Sync(lblis *cloudprovider.SLoadbalancerListener) error {
	return listerner.lb.region.SyncLoadbalancerHTTPSListener(listerner.lb, lblis)
}

func (listerner *SLoadbalancerHTTPSListener) GetProjectId() string {
	return ""
}
