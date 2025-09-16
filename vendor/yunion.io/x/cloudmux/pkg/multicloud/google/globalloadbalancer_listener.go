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

type SGlobalLoadbalancerListener struct {
	lb             *SGlobalLoadbalancer
	rules          []SGlobalLoadbalancerListenerRule
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

func (self *SGlobalLoadbalancerListener) GetId() string {
	return fmt.Sprintf("%s::%s::%s", self.forwardRule.GetGlobalId(), self.backendService.GetGlobalId(), self.Port)
}

func (self *SGlobalLoadbalancerListener) GetName() string {
	return fmt.Sprintf("%s::%s::%s", self.forwardRule.GetName(), self.backendService.GetName(), self.Port)
}

func (self *SGlobalLoadbalancerListener) GetGlobalId() string {
	return self.GetId()
}

func (self *SGlobalLoadbalancerListener) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SGlobalLoadbalancerListener) GetDescription() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SGlobalLoadbalancerListener) Refresh() error {
	return nil
}

func (self *SGlobalLoadbalancerListener) IsEmulated() bool {
	return true
}

func (self *SGlobalLoadbalancerListener) GetSysTags() map[string]string {
	return nil
}

func (self *SGlobalLoadbalancerListener) GetTags() (map[string]string, error) {
	if len(self.forwardRule.IPAddress) > 0 {
		return map[string]string{"FrontendIP": self.forwardRule.IPAddress}, nil
	}

	return map[string]string{}, nil
}

func (self *SGlobalLoadbalancerListener) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancerListener) GetListenerType() string {
	return self.Protocol
}

func (self *SGlobalLoadbalancerListener) GetListenerPort() int {
	port, err := strconv.Atoi(self.Port)
	if err != nil {
		log.Errorf("GetListenerPort %s", err)
		return 0
	}

	return port
}

func (self *SGlobalLoadbalancerListener) GetScheduler() string {
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

func (self *SGlobalLoadbalancerListener) GetAclStatus() string {
	return api.LB_BOOL_OFF
}

func (self *SGlobalLoadbalancerListener) GetAclType() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) GetAclId() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) GetEgressMbps() int {
	return 0
}

func (self *SGlobalLoadbalancerListener) GetBackendGroupId() string {
	return self.backendService.GetGlobalId()
}

func (self *SGlobalLoadbalancerListener) GetClientIdleTimeout() int {
	return int(self.backendService.ConnectionDraining.DrainingTimeoutSEC)
}

func (self *SGlobalLoadbalancerListener) GetBackendConnectTimeout() int {
	return int(self.backendService.TimeoutSEC)
}

func (self *SGlobalLoadbalancerListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SGlobalLoadbalancerListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
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

func (self *SGlobalLoadbalancerListener) GetStickySession() string {
	if self.backendService.SessionAffinity == "NONE" {
		return api.LB_BOOL_OFF
	} else {
		return api.LB_BOOL_ON
	}
}

func (self *SGlobalLoadbalancerListener) GetStickySessionType() string {
	switch self.backendService.SessionAffinity {
	case "HTTP_COOKIE":
		return api.LB_STICKY_SESSION_TYPE_SERVER
	case "GENERATED_COOKIE":
		return api.LB_STICKY_SESSION_TYPE_INSERT
	}
	return self.backendService.SessionAffinity
}

func (self *SGlobalLoadbalancerListener) GetStickySessionCookie() string {
	return self.backendService.ConsistentHash.HTTPCookie.Name
}

func (self *SGlobalLoadbalancerListener) GetStickySessionCookieTimeout() int {
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

func (self *SGlobalLoadbalancerListener) XForwardedForEnabled() bool {
	return true
}

func (self *SGlobalLoadbalancerListener) GzipEnabled() bool {
	return false
}

func (self *SGlobalLoadbalancerListener) GetCertificateId() string {
	if self.httpsProxy != nil && len(self.httpsProxy.SSLCertificates) > 0 {
		cert := SResourceBase{
			Name:     "",
			SelfLink: self.httpsProxy.SSLCertificates[0],
		}
		return cert.GetGlobalId()
	}

	return ""
}

func (self *SGlobalLoadbalancerListener) GetTLSCipherPolicy() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) HTTP2Enabled() bool {
	return true
}

func (self *SGlobalLoadbalancerListener) GetRedirect() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) GetRedirectCode() int64 {
	return 0
}

func (self *SGlobalLoadbalancerListener) GetRedirectScheme() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) GetRedirectHost() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) GetRedirectPath() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) GetHealthCheck() string {
	if len(self.backendService.HealthChecks) > 0 {
		return api.LB_BOOL_ON
	} else {
		return api.LB_BOOL_OFF
	}
}

func (self *SGlobalLoadbalancerListener) GetHealthCheckTimeout() int {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return 0
	}

	return int(hcs[0].TimeoutSEC)
}

func (self *SGlobalLoadbalancerListener) GetHealthCheckInterval() int {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return 0
	}

	return int(hcs[0].CheckIntervalSEC)
}

func (self *SGlobalLoadbalancerListener) GetHealthCheckRise() int {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return 0
	}

	return int(hcs[0].HealthyThreshold)
}

func (self *SGlobalLoadbalancerListener) GetHealthCheckFail() int {
	hcs := self.GetHealthChecks()
	if hcs == nil {
		return 0
	}

	return int(hcs[0].UnhealthyThreshold)
}

func (self *SGlobalLoadbalancerListener) GetHealthCheckReq() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) GetHealthCheckExp() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) GetHealthCheckDomain() string {
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

func (self *SGlobalLoadbalancerListener) GetHealthCheckURI() string {
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

func (self *SGlobalLoadbalancerListener) GetHealthCheckCode() string {
	return ""
}

func (self *SGlobalLoadbalancerListener) Start() error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancerListener) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancerListener) ChangeScheduler(ctx context.Context, opts *cloudprovider.ChangeListenerSchedulerOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SGlobalLoadbalancerListener) SetHealthCheck(ctx context.Context, opts *cloudprovider.ListenerHealthCheckOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SGlobalLoadbalancerListener) ChangeCertificate(ctx context.Context, opts *cloudprovider.ListenerCertificateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancerListener) SetAcl(ctx context.Context, opts *cloudprovider.ListenerAclOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SGlobalLoadbalancerListener) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadbalancer) GetLoadbalancerListeners() ([]SGlobalLoadbalancerListener, error) {
	if self.urlMap != nil {
		return self.GetHTTPLoadbalancerListeners()
	} else {
		return self.GetNetworkLoadbalancerListeners()
	}
}

func (self *SGlobalLoadbalancer) GetHTTPLoadbalancerListeners() ([]SGlobalLoadbalancerListener, error) {
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

	lbls := make([]SGlobalLoadbalancerListener, 0)
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

			lbl := SGlobalLoadbalancerListener{
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

func (self *SGlobalLoadbalancer) GetNetworkLoadbalancerListeners() ([]SGlobalLoadbalancerListener, error) {
	frs, err := self.GetForwardingRules()
	if err != nil {
		return nil, errors.Wrap(err, "GetForwardingRules")
	}

	bss, err := self.GetBackendServices()
	if err != nil {
		return nil, errors.Wrap(err, "GetBackendServices")
	}

	lbls := make([]SGlobalLoadbalancerListener, 0)
	for i := range frs {
		fr := frs[i]
		for j := range bss {
			bs := bss[j]
			if fr.Ports == nil || len(fr.Ports) == 0 {
				ports := strings.Split(fr.PortRange, "-")
				if len(ports) == 2 && ports[0] == ports[1] {
					lbl := SGlobalLoadbalancerListener{
						lb:                 self,
						forwardRule:        fr,
						backendService:     bs,
						ForwardRuleName:    fr.GetName(),
						BackendServiceName: bs.GetName(),
						Protocol:           strings.ToLower(fr.IPProtocol),
						Port:               ports[0],
					}

					lbls = append(lbls, lbl)
				}
			} else {
				for n := range fr.Ports {
					lbl := SGlobalLoadbalancerListener{
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
	}

	return lbls, nil
}

func (self *SGlobalLoadbalancerListener) GetHealthChecks() []HealthChecks {
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

func (self *SGlobalLoadbalancerListener) GetHealthCheckType() string {
	hcs := self.GetHealthChecks()
	if len(hcs) == 0 {
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

func (self *SGlobalLoadbalancerListener) GetBackendServerPort() int {
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

func (self *SGlobalLoadbalancerListener) GetInstanceGroups() ([]SGlobalInstanceGroup, error) {
	igs, err := self.lb.GetInstanceGroupsMap()
	if err != nil {
		return nil, errors.Wrap(err, "GetInstanceGroups")
	}

	ret := make([]SGlobalInstanceGroup, 0)
	for i := range self.backendService.Backends {
		b := self.backendService.Backends[i]
		if ig, ok := igs[b.Group]; ok {
			ret = append(ret, ig)
		}
	}

	return ret, nil
}

func (self *SGlobalLoadbalancerListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
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

func (self *SGlobalLoadbalancerListener) GetLoadbalancerListenerRules() ([]SGlobalLoadbalancerListenerRule, error) {
	if !self.lb.isHttpLb {
		return nil, nil
	}

	if self.rules != nil {
		return self.rules, nil
	}

	hostRules := self.lb.urlMap.HostRules
	pathMatchers := self.lb.urlMap.PathMatchers

	pmm := make(map[string]PathMatcher, 0)
	for i := range pathMatchers {
		name := pathMatchers[i].Name
		pmm[name] = pathMatchers[i]
	}

	ret := make([]SGlobalLoadbalancerListenerRule, 0)
	for _, rule := range hostRules {
		pm, ok := pmm[rule.PathMatcher]
		if !ok {
			continue
		}

		for i := range rule.Hosts {
			host := rule.Hosts[i]
			for j := range pm.PathRules {
				pr := pm.PathRules[j]

				if pr.Service != self.backendService.GetId() {
					continue
				}

				r := SGlobalLoadbalancerListenerRule{
					lbl:                self,
					backendService:     self.backendService,
					BackendServiceName: self.backendService.GetName(),
					pathMatcher:        pm,
					pathRule:           pr,
					ListenerName:       self.GetName(),
					Domain:             host,
					Path:               strings.Join(pr.Paths, ","),
					Port:               self.Port,
				}

				ret = append(ret, r)
			}
		}
	}

	self.rules = ret
	return ret, nil
}
