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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type HttpListenerProperties struct {
	ProvisioningState       string
	FrontendIPConfiguration struct {
		Id string
	}
	FrontendPort struct {
		Id string
	}
	Protocol            string
	RequestRoutingRules []struct {
		Id string
	}
	Probe struct {
		Id string
	}
}

type SLoadBalancerHTTPListener struct {
	multicloud.SResourceBase
	AzureTags
	multicloud.SLoadbalancerRedirectBase
	lb *SLoadbalancer

	Name       string                 `json:"name"`
	Id         string                 `json:"id"`
	Etag       string                 `json:"etag"`
	Type       string                 `json:"type"`
	Properties HttpListenerProperties `json:"properties"`
}

func (self *SLoadBalancerHTTPListener) GetId() string {
	return self.Id
}

func (self *SLoadBalancerHTTPListener) GetName() string {
	return self.Name
}

func (self *SLoadBalancerHTTPListener) GetGlobalId() string {
	return strings.ToLower(self.GetId())
}

func (self *SLoadBalancerHTTPListener) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "Succeeded", "Updating", "Deleting":
		return api.LB_STATUS_ENABLED
	case "Failed":
		return api.LB_STATUS_DISABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadBalancerHTTPListener) Refresh() error {
	lblis, err := self.lb.GetILoadBalancerListenerById(self.GetId())
	if err != nil {
		return errors.Wrap(err, "GetILoadBalancerListenerById")
	}

	return jsonutils.Update(self, lblis)
}

func (listerner *SLoadBalancerHTTPListener) ChangeCertificate(ctx context.Context, opts *cloudprovider.ListenerCertificateOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (listerner *SLoadBalancerHTTPListener) SetAcl(ctx context.Context, opts *cloudprovider.ListenerAclOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SLoadBalancerHTTPListener) GetProjectId() string {
	return getResourceGroup(self.GetId())
}

func (self *SLoadBalancerHTTPListener) GetListenerType() string {
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

func (self *SLoadBalancerHTTPListener) GetListenerPort() int {
	for _, port := range self.lb.Properties.FrontendPorts {
		if port.ID == self.Properties.FrontendPort.Id {
			return port.Properties.Port
		}
	}
	return 0
}

func (self *SLoadBalancerHTTPListener) GetScheduler() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetAclStatus() string {
	return api.LB_BOOL_OFF
}

func (self *SLoadBalancerHTTPListener) GetAclType() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetAclId() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetEgressMbps() int {
	return 0
}

func (self *SLoadBalancerHTTPListener) GetHealthCheck() string {
	if len(self.Properties.Probe.Id) > 0 {
		return api.LB_BOOL_ON
	}
	return api.LB_BOOL_OFF
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckType() string {
	for _, prob := range self.lb.Properties.Probes {
		if strings.EqualFold(prob.Id, self.Properties.Probe.Id) {
			return strings.ToLower(prob.Properties.Protocol)
		}
	}
	return ""
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckTimeout() int {
	return 0
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckInterval() int {
	for _, prob := range self.lb.Properties.Probes {
		if strings.EqualFold(prob.Id, self.Properties.Probe.Id) {
			return prob.Properties.IntervalInSeconds
		}
	}
	return 0
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckRise() int {
	return 0
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckFail() int {
	for _, prob := range self.lb.Properties.Probes {
		if strings.EqualFold(prob.Id, self.Properties.Probe.Id) {
			return prob.Properties.NumberOfProbes
		}
	}
	return 0
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckReq() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckExp() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetBackendGroupId() string {
	for _, rule := range self.lb.Properties.RequestRoutingRules {
		if strings.EqualFold(rule.Properties.HTTPListener.ID, self.Id) {
			return strings.ToLower(rule.Properties.BackendAddressPool.ID)
		}
	}
	return ""
}

func (self *SLoadBalancerHTTPListener) GetBackendServerPort() int {
	return 0
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckDomain() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckURI() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetHealthCheckCode() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotImplemented, "CreateILoadBalancerListenerRule")
}

func (self *SLoadBalancerHTTPListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
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

func (self *SLoadBalancerHTTPListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return []cloudprovider.ICloudLoadbalancerListenerRule{}, nil
}

func (self *SLoadBalancerHTTPListener) GetStickySession() string {
	return api.LB_BOOL_OFF
}

func (self *SLoadBalancerHTTPListener) GetStickySessionType() string {
	if self.GetStickySession() == api.LB_BOOL_ON {
		return api.LB_STICKY_SESSION_TYPE_INSERT
	}
	return ""
}

func (self *SLoadBalancerHTTPListener) GetStickySessionCookie() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetStickySessionCookieTimeout() int {
	return 0
}

func (self *SLoadBalancerHTTPListener) XForwardedForEnabled() bool {
	return false
}

func (self *SLoadBalancerHTTPListener) GzipEnabled() bool {
	return false
}

func (self *SLoadBalancerHTTPListener) GetCertificateId() string {
	for _, cert := range self.lb.Properties.SSLCertificates {
		for _, lis := range cert.Properties.HttpListeners {
			if strings.EqualFold(lis.Id, self.Id) {
				return cert.GetGlobalId()
			}
		}
	}
	return ""
}

func (self *SLoadBalancerHTTPListener) GetTLSCipherPolicy() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) HTTP2Enabled() bool {
	return self.lb.Properties.EnableHttp2
}

func (self *SLoadBalancerHTTPListener) Start() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Start")
}

func (self *SLoadBalancerHTTPListener) Stop() error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Stop")
}

func (self *SLoadBalancerHTTPListener) ChangeScheduler(ctx context.Context, opts *cloudprovider.ChangeListenerSchedulerOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SLoadBalancerHTTPListener) SetHealthCheck(ctx context.Context, opts *cloudprovider.ListenerHealthCheckOptions) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SLoadBalancerHTTPListener) Delete(ctx context.Context) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SLoadBalancerHTTPListener) GetRedirect() string {
	return api.LB_REDIRECT_OFF
}

func (self *SLoadBalancerHTTPListener) GetRedirectCode() int64 {
	return 0
}

func (self *SLoadBalancerHTTPListener) GetRedirectScheme() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetRedirectHost() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetRedirectPath() string {
	return ""
}

func (self *SLoadBalancerHTTPListener) GetClientIdleTimeout() int {
	return 0
}

func (self *SLoadBalancerHTTPListener) GetBackendConnectTimeout() int {
	return 0
}
