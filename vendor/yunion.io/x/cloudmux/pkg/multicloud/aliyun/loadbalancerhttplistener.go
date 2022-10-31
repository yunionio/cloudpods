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
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancerHTTPListener struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	AliyunTags
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

	AclType string //	访问控制类型：
	//white： 仅转发来自所选访问控制策略组中设置的IP地址或地址段的请求，白名单适用于应用只允许特定IP访问的场景。
	//设置白名单存在一定业务风险。一旦设置白名单，就只有白名单中的IP可以访问负载均衡监听。如果开启了白名单访问，但访问策略组中没有添加任何IP，则负载均衡监听会转发全部请求。

	//black： 来自所选访问控制策略组中设置的IP地址或地址段的所有请求都不会转发，黑名单适用于应用只限制某些特定IP访问的场景。
	//如果开启了黑名单访问，但访问策略组中没有添加任何IP，则负载均衡监听会转发全部请求。

	//当AclStatus参数的值为on时，该参数必选。

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
	Gzip                   string //	是否开启Gzip压缩。
	EnableHttp2            string //	是否开启HTTP/2特性。取值：on（默认值）|off

	Rules           Rules  //监听下的转发规则列表，具体请参见RuleList。
	ForwardPort     int    //	HTTP至HTTPS的监听转发端口。暂时只支持将HTTP 80访问重定向转发至HTTPS 443。 说明 如果 ListenerForward的值为 off，该参数不显示。
	ListenerForward string //	表示是否开启HTTP至HTTPS的监听转发。on：表示开启 off：表示未开启
	VServerGroupId  string // 绑定的服务器组ID
}

func (listener *SLoadbalancerHTTPListener) GetName() string {
	if len(listener.Description) == 0 {
		listener.Refresh()
	}
	if len(listener.Description) > 0 {
		return listener.Description
	}
	return fmt.Sprintf("HTTP:%d", listener.ListenerPort)
}

func (listerner *SLoadbalancerHTTPListener) GetId() string {
	return fmt.Sprintf("%s/%d", listerner.lb.LoadBalancerId, listerner.ListenerPort)
}

func (listerner *SLoadbalancerHTTPListener) GetGlobalId() string {
	return listerner.GetId()
}

func (listerner *SLoadbalancerHTTPListener) GetStatus() string {
	switch listerner.Status {
	case "starting", "running":
		return api.LB_STATUS_ENABLED
	case "configuring", "stopping", "stopped":
		return api.LB_STATUS_DISABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (listerner *SLoadbalancerHTTPListener) IsEmulated() bool {
	return false
}

func (listerner *SLoadbalancerHTTPListener) GetEgressMbps() int {
	if listerner.Bandwidth < 1 {
		return 0
	}
	return listerner.Bandwidth
}

func (listerner *SLoadbalancerHTTPListener) Refresh() error {
	lis, err := listerner.lb.region.GetLoadbalancerHTTPListener(listerner.lb.LoadBalancerId, listerner.ListenerPort)
	if err != nil {
		return err
	}
	return jsonutils.Update(listerner, lis)
}

func (listerner *SLoadbalancerHTTPListener) GetListenerType() string {
	return "http"
}

func (listerner *SLoadbalancerHTTPListener) GetListenerPort() int {
	return listerner.ListenerPort
}

func (listerner *SLoadbalancerHTTPListener) GetBackendGroupId() string {
	if len(listerner.VServerGroupId) == 0 {
		listerner.Refresh()
	}
	return listerner.VServerGroupId
}

func (listerner *SLoadbalancerHTTPListener) GetBackendServerPort() int {
	return listerner.BackendServerPort
}

func (listerner *SLoadbalancerHTTPListener) GetScheduler() string {
	return listerner.Scheduler
}

func (listerner *SLoadbalancerHTTPListener) GetAclStatus() string {
	return listerner.AclStatus
}

func (listerner *SLoadbalancerHTTPListener) GetAclType() string {
	return listerner.AclType
}

func (listerner *SLoadbalancerHTTPListener) GetAclId() string {
	return listerner.AclId
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheck() string {
	return listerner.HealthCheck
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckType() string {
	return api.LB_HEALTH_CHECK_HTTP
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckDomain() string {
	return listerner.HealthCheckDomain
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckURI() string {
	return listerner.HealthCheckURI
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckCode() string {
	return listerner.HealthCheckHttpCode
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckRise() int {
	return listerner.HealthyThreshold
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckFail() int {
	return listerner.UnhealthyThreshold
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckTimeout() int {
	return listerner.HealthCheckTimeout
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckInterval() int {
	return listerner.HealthCheckInterval
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckReq() string {
	return ""
}

func (listerner *SLoadbalancerHTTPListener) GetHealthCheckExp() string {
	return ""
}

func (listerner *SLoadbalancerHTTPListener) GetStickySession() string {
	return listerner.StickySession
}

func (listerner *SLoadbalancerHTTPListener) GetStickySessionType() string {
	return listerner.StickySessionType
}

func (listerner *SLoadbalancerHTTPListener) GetStickySessionCookie() string {
	return listerner.Cookie
}

func (listerner *SLoadbalancerHTTPListener) GetStickySessionCookieTimeout() int {
	return listerner.CookieTimeout
}

func (listerner *SLoadbalancerHTTPListener) XForwardedForEnabled() bool {
	if listerner.XForwardedFor == "on" {
		return true
	}
	return false
}

func (listerner *SLoadbalancerHTTPListener) GzipEnabled() bool {
	if listerner.Gzip == "on" {
		return true
	}
	return false
}

func (listerner *SLoadbalancerHTTPListener) GetCertificateId() string {
	return ""
}

func (listerner *SLoadbalancerHTTPListener) GetTLSCipherPolicy() string {
	return ""
}

func (listerner *SLoadbalancerHTTPListener) HTTP2Enabled() bool {
	if listerner.EnableHttp2 == "on" {
		return true
	}
	return false
}

func (listerner *SLoadbalancerHTTPListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := listerner.lb.region.GetLoadbalancerListenerRules(listerner.lb.LoadBalancerId, listerner.ListenerPort)
	if err != nil {
		return nil, err
	}
	iRules := []cloudprovider.ICloudLoadbalancerListenerRule{}
	for i := 0; i < len(rules); i++ {
		rules[i].httpListener = listerner
		iRules = append(iRules, &rules[i])
	}
	return iRules, nil
}

func (region *SRegion) GetLoadbalancerHTTPListener(loadbalancerId string, listenerPort int) (*SLoadbalancerHTTPListener, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["ListenerPort"] = fmt.Sprintf("%d", listenerPort)
	body, err := region.lbRequest("DescribeLoadBalancerHTTPListenerAttribute", params)
	if err != nil {
		return nil, err
	}
	listener := SLoadbalancerHTTPListener{}
	return &listener, body.Unmarshal(&listener)
}

func (region *SRegion) DeleteLoadbalancerListener(loadbalancerId string, listenerPort int) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["ListenerPort"] = fmt.Sprintf("%d", listenerPort)
	_, err := region.lbRequest("DeleteLoadBalancerListener", params)
	return err
}

func (region *SRegion) CreateLoadbalancerHTTPListener(lb *SLoadbalancer, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	params := region.constructBaseCreateListenerParams(lb, listener)
	params = region.constructHTTPCreateListenerParams(params, listener)
	_, err := region.lbRequest("CreateLoadBalancerHTTPListener", params)
	if err != nil {
		return nil, err
	}
	iListener, err := region.GetLoadbalancerHTTPListener(lb.LoadBalancerId, listener.ListenerPort)
	if err != nil {
		return nil, err
	}
	iListener.lb = lb
	return iListener, nil
}

func (listerner *SLoadbalancerHTTPListener) Delete(ctx context.Context) error {
	return listerner.lb.region.DeleteLoadbalancerListener(listerner.lb.LoadBalancerId, listerner.ListenerPort)
}

func (listerner *SLoadbalancerHTTPListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	_rule := &SLoadbalancerListenerRule{
		Domain:   rule.Domain,
		Url:      rule.Path,
		RuleName: rule.Name,
	}
	if len(rule.BackendGroupId) > 0 { //&& rule.BackendGroupType == api.LB_BACKENDGROUP_TYPE_NORMAL {
		_rule.VServerGroupId = rule.BackendGroupId
	}
	listenerRule, err := listerner.lb.region.CreateLoadbalancerListenerRule(listerner.ListenerPort, listerner.lb.LoadBalancerId, _rule)
	if err != nil {
		return nil, err
	}
	listenerRule.httpListener = listerner
	return listenerRule, nil
}

func (listerner *SLoadbalancerHTTPListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rule, err := listerner.lb.region.GetLoadbalancerListenerRule(ruleId)
	if err != nil {
		return nil, err
	}
	rule.httpListener = listerner
	return rule, nil
}

func (region *SRegion) startListener(listenerPort int, loadbalancerId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["ListenerPort"] = fmt.Sprintf("%d", listenerPort)
	_, err := region.lbRequest("StartLoadBalancerListener", params)
	return err
}

func (region *SRegion) stopListener(listenerPort int, loadbalancerId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["ListenerPort"] = fmt.Sprintf("%d", listenerPort)
	_, err := region.lbRequest("StopLoadBalancerListener", params)
	return err
}

func (listerner *SLoadbalancerHTTPListener) Start() error {
	return listerner.lb.region.startListener(listerner.ListenerPort, listerner.lb.LoadBalancerId)
}

func (listerner *SLoadbalancerHTTPListener) Stop() error {
	return listerner.lb.region.stopListener(listerner.ListenerPort, listerner.lb.LoadBalancerId)
}

func (region *SRegion) SyncLoadbalancerHTTPListener(lb *SLoadbalancer, listener *cloudprovider.SLoadbalancerListenerCreateOptions) error {
	params := region.constructBaseCreateListenerParams(lb, listener)
	params = region.constructHTTPCreateListenerParams(params, listener)
	_, err := region.lbRequest("SetLoadBalancerHTTPListenerAttribute", params)
	return err
}

func (listerner *SLoadbalancerHTTPListener) Sync(ctx context.Context, lblis *cloudprovider.SLoadbalancerListenerCreateOptions) error {
	return listerner.lb.region.SyncLoadbalancerHTTPListener(listerner.lb, lblis)
}

func (listerner *SLoadbalancerHTTPListener) GetProjectId() string {
	return listerner.lb.GetProjectId()
}

func (listerner *SLoadbalancerHTTPListener) GetClientIdleTimeout() int {
	return 0
}

func (listerner *SLoadbalancerHTTPListener) GetBackendConnectTimeout() int {
	return 0
}
