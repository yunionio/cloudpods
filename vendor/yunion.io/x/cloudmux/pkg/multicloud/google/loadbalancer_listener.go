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

package google

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SLoadbalancerListener struct {
	lb             *SLoadbalancer
	rules          []SLoadbalancerListenerRule
	forwardRule    SForwardingRule    // 服务IP地址
	backendService SBackendServices   //
	httpProxy      *STargetHttpProxy  // http
	httpsProxy     *STargetHttpsProxy // https
	healthChecks   []HealthChecks

	ForwardRuleName    string `json:"forward_rule_name"`
	BackendServiceName string `json:"backend_service_name"`
	Protocol           string `json:"protocol"`
	Port               string `json:"port"` // 监听端口
}

func (self *SLoadbalancerListener) GetId() string {
	return fmt.Sprintf("%s::%s::%s", self.forwardRule.GetGlobalId(), self.backendService.GetGlobalId(), self.Port)
}

func (self *SLoadbalancerListener) GetName() string {
	return fmt.Sprintf("%s::%s::%s", self.forwardRule.GetName(), self.backendService.GetName(), self.Port)
}

func (self *SLoadbalancerListener) GetGlobalId() string {
	return self.GetId()
}

func (self *SLoadbalancerListener) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancerListener) Refresh() error {
	return nil
}

func (self *SLoadbalancerListener) IsEmulated() bool {
	return true
}

func (self *SLoadbalancerListener) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SLoadbalancerListener) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadbalancerListener) GetTags() (map[string]string, error) {
	if len(self.forwardRule.IPAddress) > 0 {
		return map[string]string{"FrontendIP": self.forwardRule.IPAddress}, nil
	}

	return map[string]string{}, nil
}

func (self *SLoadbalancerListener) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerListener) GetProjectId() string {
	return self.lb.GetProjectId()
}

func (self *SLoadbalancerListener) GetListenerType() string {
	return self.Protocol
}

func (self *SLoadbalancerListener) GetListenerPort() int {
	port, err := strconv.Atoi(self.Port)
	if err != nil {
		log.Errorf("GetListenerPort %s", err)
		return 0
	}

	return port
}

/*
在本地范围内使用的负载均衡算法。可能的值为：

ROUND_ROBIN：这是一个简单的策略，其中按循环顺序选择每个健康的后端。这是默认设置。
LEAST_REQUEST：一种 O(1) 算法，它选择两个随机的健康主机并选择具有较少活动请求的主机。
RING_HASH：环/模散列负载均衡器对后端实现一致的散列。该算法的特性是从一组 N 个主机中添加/删除一个主机只会影响 1/N 的请求。
RANDOM：负载均衡器随机选择一个健康的主机。
ORIGINAL_DESTINATION：根据客户端连接元数据选择后端主机，即在连接被重定向到负载均衡器之前，连接被打开到与传入连接的目标地址相同的地址。
MAGLEV：用作环形哈希负载均衡器的替代品。Maglev 不如环哈希稳定，但具有更快的表查找构建时间和主机选择时间。有关磁悬浮的更多信息，请参阅https://ai.google/research/pubs/pub44824
此字段适用于：

service_protocol 设置为 HTTP、HTTPS 或 HTTP2，并且 load_balancing_scheme 设置为 INTERNAL_MANAGED 的区域后端服务。
load_balancing_scheme 设置为 INTERNAL_SELF_MANAGED 的全局后端服务。
如果 sessionAffinity 不为 NONE，并且该字段未设置为 MAGLEV 或 RING_HASH，则会话亲缘性设置不会生效。

当后端服务被绑定到目标 gRPC 代理且 validateForProxyless 字段设置为 true 的 URL 映射引用时，仅支持默认的 ROUND_ROBIN 策略。
*/
// todo: fix me ???
func (self *SLoadbalancerListener) GetScheduler() string {
	switch self.backendService.LocalityLBPolicy {
	case "ROUND_ROBIN":
		return api.LB_SCHEDULER_RR
	case "LEAST_REQUEST":
		return api.LB_SCHEDULER_WLC
	case "RING_HASH":
		return api.LB_SCHEDULER_QCH
	case "ORIGINAL_DESTINATION":
		return api.LB_SCHEDULER_SCH
	case "MAGLEV":
		return api.LB_SCHEDULER_MH
	default:
		return ""
	}
}

func (self *SLoadbalancerListener) GetAclStatus() string {
	return api.LB_BOOL_OFF
}

func (self *SLoadbalancerListener) GetAclType() string {
	return ""
}

func (self *SLoadbalancerListener) GetAclId() string {
	return ""
}

func (self *SLoadbalancerListener) GetEgressMbps() int {
	return 0
}

func (self *SLoadbalancerListener) GetBackendGroupId() string {
	return self.backendService.GetGlobalId()
}

func (self *SLoadbalancerListener) GetBackendServerPort() int {
	igs, err := self.GetInstanceGroups()
	if err != nil {
		log.Errorf("GetInstanceGroups %s", err)
		return 0
	}

	for i := range igs {
		for j := range igs[i].NamedPorts {
			if igs[i].NamedPorts[j].Name == self.backendService.PortName {
				return int(igs[i].NamedPorts[j].Port)
			}
		}
	}

	return 0
}

func (self *SLoadbalancerListener) GetClientIdleTimeout() int {
	return int(self.backendService.ConnectionDraining.DrainingTimeoutSEC)
}

func (self *SLoadbalancerListener) GetBackendConnectTimeout() int {
	return int(self.backendService.TimeoutSEC)
}

func (self *SLoadbalancerListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SLoadbalancerListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := self.GetLoadbalancerListenerRules()
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancerListenerRules")
	}

	for i := range rules {
		if rules[i].GetGlobalId() == ruleId {
			return &rules[i], nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SLoadbalancerListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := self.GetLoadbalancerListenerRules()
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancerListenerRules")
	}

	irules := make([]cloudprovider.ICloudLoadbalancerListenerRule, len(rules))
	for i := range rules {
		irules[i] = &rules[i]
	}

	return irules, nil
}

func (self *SLoadbalancerListener) GetStickySession() string {
	if self.backendService.SessionAffinity == "NONE" {
		return api.LB_BOOL_OFF
	} else {
		return api.LB_BOOL_ON
	}
}

/*
https://cloud.google.com/load-balancing/docs/backend-service#sessionAffinity
区域级外部 HTTP(S) 负载均衡器:
   无 (NONE)
   客户端 IP (CLIENT_IP)
   生成的 Cookie (GENERATED_COOKIE)
   标头字段 (HEADER_FIELD)
   HTTP Cookie (HTTP_COOKIE)
*/
func (self *SLoadbalancerListener) GetStickySessionType() string {
	switch self.backendService.SessionAffinity {
	case "HTTP_COOKIE":
		return api.LB_STICKY_SESSION_TYPE_SERVER
	case "GENERATED_COOKIE":
		return api.LB_STICKY_SESSION_TYPE_INSERT
	}
	return self.backendService.SessionAffinity
}

func (self *SLoadbalancerListener) GetStickySessionCookie() string {
	return self.backendService.ConsistentHash.HTTPCookie.Name
}

func (self *SLoadbalancerListener) GetStickySessionCookieTimeout() int {
	if len(self.backendService.ConsistentHash.HTTPCookie.TTL.Seconds) == 0 {
		return 0
	}

	sec, err := strconv.Atoi(self.backendService.ConsistentHash.HTTPCookie.TTL.Seconds)
	if err != nil {
		log.Debugf("GetStickySessionCookieTimeout %s", err)
		return 0
	}

	return sec
}

// https://cloud.google.com/load-balancing/docs/https
func (self *SLoadbalancerListener) XForwardedForEnabled() bool {
	return true
}

// https://cloud.google.com/load-balancing/docs/https/troubleshooting-ext-https-lbs
func (self *SLoadbalancerListener) GzipEnabled() bool {
	return false
}

func (self *SLoadbalancerListener) GetCertificateId() string {
	if self.httpsProxy != nil && len(self.httpsProxy.SSLCertificates) > 0 {
		cert := SResourceBase{
			Name:     "",
			SelfLink: self.httpsProxy.SSLCertificates[0],
		}
		return cert.GetGlobalId()
	}

	return ""
}

func (self *SLoadbalancerListener) GetTLSCipherPolicy() string {
	return ""
}

// https://cloud.google.com/load-balancing/docs/https/troubleshooting-ext-https-lbs
func (self *SLoadbalancerListener) HTTP2Enabled() bool {
	return true
}

// todo: fix me route
// 高级配置才有重定向，具体怎么解析？
func (self *SLoadbalancerListener) GetRedirect() string {
	//self.lb.urlMap.PathMatchers[0].RouteRules[0].URLRedirect
	return ""
}

func (self *SLoadbalancerListener) GetRedirectCode() int64 {
	return 0
}

func (self *SLoadbalancerListener) GetRedirectScheme() string {
	return ""
}

func (self *SLoadbalancerListener) GetRedirectHost() string {
	return ""
}

func (self *SLoadbalancerListener) GetRedirectPath() string {
	return ""
}

func (self *SLoadbalancerListener) GetHealthCheck() string {
	if len(self.backendService.HealthChecks) > 0 {
		return api.LB_BOOL_ON
	} else {
		return api.LB_BOOL_OFF
	}
}

func (self *SLoadbalancerListener) GetHealthChecks() []HealthChecks {
	if self.healthChecks != nil {
		return self.healthChecks
	}

	hcm, err := self.lb.GetHealthCheckMaps()
	if err != nil {
		log.Errorf("GetHealthCheckMaps %s", err)
		return nil
	}
	ret := make([]HealthChecks, 0)
	for i := range self.backendService.HealthChecks {
		hc := self.backendService.HealthChecks[i]
		if _, ok := hcm[hc]; ok {
			ret = append(ret, hcm[hc])
		}
	}

	self.healthChecks = ret
	return ret
}

func (self *SLoadbalancerListener) GetHealthCheckType() string {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return ""
	}

	switch strings.ToLower(hcs[0].Type) {
	case "tcp":
		return api.LB_HEALTH_CHECK_TCP
	case "udp":
		return api.LB_HEALTH_CHECK_UDP
	case "http", "http2":
		return api.LB_HEALTH_CHECK_HTTP
	case "https", "ssl":
		return api.LB_HEALTH_CHECK_HTTPS
	default:
		return ""
	}
}

func (self *SLoadbalancerListener) GetHealthCheckTimeout() int {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return 0
	}

	return int(hcs[0].TimeoutSEC)
}

func (self *SLoadbalancerListener) GetHealthCheckInterval() int {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return 0
	}

	return int(hcs[0].CheckIntervalSEC)
}

func (self *SLoadbalancerListener) GetHealthCheckRise() int {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return 0
	}

	return int(hcs[0].HealthyThreshold)
}

func (self *SLoadbalancerListener) GetHealthCheckFail() int {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return 0
	}

	return int(hcs[0].UnhealthyThreshold)
}

func (self *SLoadbalancerListener) GetHealthCheckReq() string {
	return ""
}

func (self *SLoadbalancerListener) GetHealthCheckExp() string {
	return ""
}

func (self *SLoadbalancerListener) GetHealthCheckDomain() string {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return ""
	}
	switch hcs[0].Type {
	case "HTTPS":
		return hcs[0].HTTPSHealthCheck.Host
	case "HTTP2":
		return hcs[0].Http2HealthCheck.Host
	case "HTTP":
		return hcs[0].HTTPHealthCheck.Host
	default:
		return ""
	}
}

func (self *SLoadbalancerListener) GetHealthCheckURI() string {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return ""
	}
	switch hcs[0].Type {
	case "HTTPS":
		return hcs[0].HTTPSHealthCheck.RequestPath
	case "HTTP2":
		return hcs[0].Http2HealthCheck.RequestPath
	case "HTTP":
		return hcs[0].HTTPHealthCheck.RequestPath
	default:
		return ""
	}
}

func (self *SLoadbalancerListener) GetHealthCheckCode() string {
	return ""
}

func (self *SLoadbalancerListener) Start() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerListener) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerListener) Sync(ctx context.Context, listener *cloudprovider.SLoadbalancerListener) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerListener) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerListener) GetInstanceGroups() ([]SInstanceGroup, error) {
	igs, err := self.lb.GetInstanceGroupsMap()
	if err != nil {
		return nil, errors.Wrap(err, "GetInstanceGroups")
	}

	ret := make([]SInstanceGroup, 0)
	for i := range self.backendService.Backends {
		b := self.backendService.Backends[i]
		if ig, ok := igs[b.Group]; ok {
			ret = append(ret, ig)
		}
	}

	return ret, nil
}

func (self *SLoadbalancer) GetLoadbalancerListeners() ([]SLoadbalancerListener, error) {
	if self.urlMap != nil {
		return self.GetHTTPLoadbalancerListeners()
	} else {
		return self.GetNetworkLoadbalancerListeners()
	}
}

func (self *SLoadbalancer) GetHTTPLoadbalancerListeners() ([]SLoadbalancerListener, error) {
	frs, err := self.GetForwardingRules()
	if err != nil {
		return nil, errors.Wrap(err, "GetForwardingRules")
	}

	_hps, err := self.GetTargetHttpProxies()
	if err != nil {
		return nil, errors.Wrap(err, "GetTargetHttpProxies")
	}

	hps := make(map[string]STargetHttpProxy, 0)
	for i := range _hps {
		hps[_hps[i].SelfLink] = _hps[i]
	}

	_hsps, err := self.GetTargetHttpsProxies()
	if err != nil {
		return nil, errors.Wrap(err, "GetTargetHttpsProxies")
	}

	hsps := make(map[string]STargetHttpsProxy, 0)
	for i := range _hsps {
		hsps[_hsps[i].SelfLink] = _hsps[i]
	}

	bss, err := self.GetBackendServices()
	if err != nil {
		return nil, errors.Wrap(err, "GetBackendServices")
	}

	lbls := make([]SLoadbalancerListener, 0)
	for i := range frs {
		fr := frs[i]
		for j := range bss {
			bs := bss[j]
			port := "80"
			protocol := "http"
			var hp STargetHttpProxy
			var hsp STargetHttpsProxy
			if fr.PortRange == "443-443" {
				port = "443"
				hsp = hsps[fr.Target]
				protocol = "https"
			} else if fr.PortRange == "8080-8080" {
				port = "8080"
				hp = hps[fr.Target]
			} else {
				hp = hps[fr.Target]
			}

			lbl := SLoadbalancerListener{
				lb:                 self,
				forwardRule:        fr,
				backendService:     bs,
				httpProxy:          &hp,
				httpsProxy:         &hsp,
				ForwardRuleName:    fr.GetName(),
				BackendServiceName: bs.GetName(),
				Protocol:           protocol,
				Port:               port,
			}
			lbls = append(lbls, lbl)
		}
	}

	return lbls, nil
}

func (self *SLoadbalancer) GetNetworkLoadbalancerListeners() ([]SLoadbalancerListener, error) {
	frs, err := self.GetForwardingRules()
	if err != nil {
		return nil, errors.Wrap(err, "GetForwardingRules")
	}

	bss, err := self.GetBackendServices()
	if err != nil {
		return nil, errors.Wrap(err, "GetBackendServices")
	}

	lbls := make([]SLoadbalancerListener, 0)
	for i := range frs {
		fr := frs[i]
		for j := range bss {
			bs := bss[j]
			for n := range fr.Ports {
				lbl := SLoadbalancerListener{
					lb:                 self,
					forwardRule:        fr,
					backendService:     bs,
					ForwardRuleName:    fr.GetName(),
					BackendServiceName: bs.GetName(),
					Protocol:           strings.ToLower(fr.IPProtocol),
					Port:               fr.Ports[n],
				}

				lbls = append(lbls, lbl)
			}
		}
	}

	return lbls, nil
}
