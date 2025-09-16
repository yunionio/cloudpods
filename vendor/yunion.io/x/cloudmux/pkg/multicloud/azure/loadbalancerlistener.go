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

package azure

import (
	"context"
	"strings"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type ListenerProperties struct {
	ProvisioningState       string `json:"provisioningState"`
	FrontendIPConfiguration Subnet `json:"frontendIPConfiguration"`
	FrontendPort            int    `json:"frontendPort"`
	BackendPort             int    `json:"backendPort"`
	AllocatedOutboundPorts  int    `json:"allocatedOutboundPorts"`
	EnableFloatingIP        bool   `json:"enableFloatingIP"`
	IdleTimeoutInMinutes    int64  `json:"idleTimeoutInMinutes"`
	Protocol                string `json:"protocol"`
	EnableTCPReset          bool   `json:"enableTcpReset"`
	LoadDistribution        string
	BackendIPConfiguration  struct {
		Id string `json:"id"`
	} `json:"backendIPConfiguration"`
	BackendAddressPool struct {
		Id string `json:"id"`
	} `json:"backendAddressPool"`
	BackendAddressPools []struct {
		Id string `json:"id"`
	} `json:"backendAddressPools"`
	Probe struct {
		Id string `json:"id"`
	}
}

type SLoadBalancerListener struct {
	multicloud.SResourceBase
	AzureTags
	multicloud.SLoadbalancerRedirectBase
	lb *SLoadbalancer

	Name       string             `json:"name"`
	Id         string             `json:"id"`
	Etag       string             `json:"etag"`
	Type       string             `json:"type"`
	Properties ListenerProperties `json:"properties"`
}

func (self *SLoadBalancerListener) GetId() string {
	return self.Id
}

func (self *SLoadBalancerListener) GetName() string {
	return self.Name
}

func (self *SLoadBalancerListener) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadBalancerListener) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "Succeeded", "Updating", "Deleting":
		return api.LB_STATUS_ENABLED
	case "Failed":
		return api.LB_STATUS_DISABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadBalancerListener) Refresh() error {
	lblis, err := self.lb.GetILoadBalancerListenerById(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetILoadBalancerListenerById")
	}

	return jsonutils.Update(self, lblis)
}

func (listerner *SLoadBalancerListener) ChangeCertificate(ctx context.Context, opts *cloudprovider.ListenerCertificateOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (listerner *SLoadBalancerListener) SetAcl(ctx context.Context, opts *cloudprovider.ListenerAclOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SLoadBalancerListener) GetProjectId() string {
	return getResourceGroup(self.GetId())
}

func (self *SLoadBalancerListener) GetListenerType() string {
	switch strings.ToLower(self.Properties.Protocol) {
	case "tcp":
		return api.LB_LISTENER_TYPE_TCP
	case "udp":
		return api.LB_LISTENER_TYPE_UDP
	case "http":
		return api.LB_LISTENER_TYPE_HTTP
	case "https":
		return api.LB_LISTENER_TYPE_HTTPS
	default:
		return strings.ToLower(self.Properties.Protocol)
	}
}

func (self *SLoadBalancerListener) GetListenerPort() int {
	return self.Properties.FrontendPort
}

func (self *SLoadBalancerListener) GetScheduler() string {
	switch self.Properties.LoadDistribution {
	case "Default":
		return api.LB_SCHEDULER_MH
	case "SourceIP":
		return api.LB_SCHEDULER_SCH
	case "SourceIPProtocol":
		return api.LB_SCHEDULER_TCH
	}
	return ""
}

func (self *SLoadBalancerListener) GetAclStatus() string {
	return api.LB_BOOL_OFF
}

func (self *SLoadBalancerListener) GetAclType() string {
	return ""
}

func (self *SLoadBalancerListener) GetAclId() string {
	return ""
}

func (self *SLoadBalancerListener) GetEgressMbps() int {
	return 0
}

func (self *SLoadBalancerListener) GetHealthCheck() string {
	if len(self.Properties.Probe.Id) > 0 {
		return api.LB_BOOL_ON
	}
	return api.LB_BOOL_OFF
}

func (self *SLoadBalancerListener) GetHealthCheckType() string {
	for _, prob := range self.lb.Properties.Probes {
		if strings.EqualFold(prob.Id, self.Properties.Probe.Id) {
			return strings.ToLower(prob.Properties.Protocol)
		}
	}
	return ""
}

func (self *SLoadBalancerListener) GetHealthCheckTimeout() int {
	return 0
}

func (self *SLoadBalancerListener) GetHealthCheckInterval() int {
	for _, prob := range self.lb.Properties.Probes {
		if strings.EqualFold(prob.Id, self.Properties.Probe.Id) {
			return prob.Properties.IntervalInSeconds
		}
	}
	return 0
}

func (self *SLoadBalancerListener) GetHealthCheckRise() int {
	return 0
}

func (self *SLoadBalancerListener) GetHealthCheckFail() int {
	for _, prob := range self.lb.Properties.Probes {
		if strings.EqualFold(prob.Id, self.Properties.Probe.Id) {
			return prob.Properties.NumberOfProbes
		}
	}
	return 0
}

func (self *SLoadBalancerListener) GetHealthCheckReq() string {
	return ""
}

func (self *SLoadBalancerListener) GetHealthCheckExp() string {
	return ""
}

func (self *SLoadBalancerListener) GetBackendGroupId() string {
	for _, id := range []string{
		self.Properties.BackendIPConfiguration.Id,
		self.Properties.BackendAddressPool.Id,
	} {
		if len(id) > 0 {
			return strings.ToLower(id)
		}
	}
	return ""
}

func (self *SLoadBalancerListener) GetBackendServerPort() int {
	return self.Properties.BackendPort
}

func (self *SLoadBalancerListener) GetHealthCheckDomain() string {
	return ""
}

func (self *SLoadBalancerListener) GetHealthCheckURI() string {
	return ""
}

func (self *SLoadBalancerListener) GetHealthCheckCode() string {
	return ""
}

func (self *SLoadBalancerListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancerListenerRule")
}

func (self *SLoadBalancerListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	lbrs, err := self.GetILoadbalancerListenerRules()
	if err != nil {
		return nil, errors.Wrap(err, "GetILoadbalancerListenerRules")
	}

	for i := range lbrs {
		if lbrs[i].GetId() == ruleId {
			return lbrs[i], nil
		}
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, ruleId)
}

func (self *SLoadBalancerListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return []cloudprovider.ICloudLoadbalancerListenerRule{}, nil
}

func (self *SLoadBalancerListener) GetStickySession() string {
	return api.LB_BOOL_OFF
}

func (self *SLoadBalancerListener) GetStickySessionType() string {
	if self.GetStickySession() == api.LB_BOOL_ON {
		return api.LB_STICKY_SESSION_TYPE_INSERT
	}
	return ""
}

func (self *SLoadBalancerListener) GetStickySessionCookie() string {
	return ""
}

func (self *SLoadBalancerListener) GetStickySessionCookieTimeout() int {
	return 0
}

func (self *SLoadBalancerListener) XForwardedForEnabled() bool {
	return false
}

func (self *SLoadBalancerListener) GzipEnabled() bool {
	return false
}

func (self *SLoadBalancerListener) GetCertificateId() string {
	return ""
}

func (self *SLoadBalancerListener) GetTLSCipherPolicy() string {
	return ""
}

func (self *SLoadBalancerListener) HTTP2Enabled() bool {
	return self.lb.Properties.EnableHttp2
}

func (self *SLoadBalancerListener) Start() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Start")
}

func (self *SLoadBalancerListener) Stop() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Stop")
}

func (self *SLoadBalancerListener) ChangeScheduler(ctx context.Context, opts *cloudprovider.ChangeListenerSchedulerOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SLoadBalancerListener) SetHealthCheck(ctx context.Context, opts *cloudprovider.ListenerHealthCheckOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SLoadBalancerListener) Delete(ctx context.Context) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadBalancerListener) GetRedirect() string {
	return api.LB_REDIRECT_OFF
}

func (self *SLoadBalancerListener) GetRedirectCode() int64 {
	return 0
}

func (self *SLoadBalancerListener) GetRedirectScheme() string {
	return ""
}

func (self *SLoadBalancerListener) GetRedirectHost() string {
	return ""
}

func (self *SLoadBalancerListener) GetRedirectPath() string {
	return ""
}

func (self *SLoadBalancerListener) GetClientIdleTimeout() int {
	return int(self.Properties.IdleTimeoutInMinutes) * 60
}

func (self *SLoadBalancerListener) GetBackendConnectTimeout() int {
	return 0
}
