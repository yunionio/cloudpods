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

type SNlbListener struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	AliyunTags
	nlb *SNlb

	ListenerId            string                 `json:"ListenerId"`
	ListenerDescription   string                 `json:"ListenerDescription"`
	ListenerProtocol      string                 `json:"ListenerProtocol"`
	ListenerPort          int                    `json:"ListenerPort"`
	ListenerStatus        string                 `json:"ListenerStatus"`
	LoadBalancerId        string                 `json:"LoadBalancerId"`
	ServerGroupId         string                 `json:"ServerGroupId"`
	IdleTimeout           int                    `json:"IdleTimeout"`
	SecurityPolicyId      string                 `json:"SecurityPolicyId"`
	CaEnabled             bool                   `json:"CaEnabled"`
	CaCertificateIds      []string               `json:"CaCertificateIds"`
	CertificateIds        []string               `json:"CertificateIds"`
	Cps                   int                    `json:"Cps"`
	Mss                   int                    `json:"Mss"`
	ProxyProtocolEnabled  bool                   `json:"ProxyProtocolEnabled"`
	SecSensorEnabled      bool                   `json:"SecSensorEnabled"`
	ProxyProtocolV2Config map[string]interface{} `json:"ProxyProtocolV2Config"`
	AlpnEnabled           bool                   `json:"AlpnEnabled"`
	AlpnPolicy            string                 `json:"AlpnPolicy"`
	StartPort             string                 `json:"StartPort"`
	EndPort               string                 `json:"EndPort"`
	RegionId              string                 `json:"RegionId"`
}

func (listener *SNlbListener) GetName() string {
	if len(listener.ListenerDescription) > 0 {
		return listener.ListenerDescription
	}
	return fmt.Sprintf("%s:%d", listener.ListenerProtocol, listener.ListenerPort)
}

func (listener *SNlbListener) GetId() string {
	return listener.ListenerId
}

func (listener *SNlbListener) GetGlobalId() string {
	return listener.ListenerId
}

func (listener *SNlbListener) GetStatus() string {
	switch listener.ListenerStatus {
	case "Running":
		return api.LB_STATUS_ENABLED
	case "Stopped":
		return api.LB_STATUS_DISABLED
	case "Provisioning", "Configuring":
		return api.LB_STATUS_UNKNOWN
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (listener *SNlbListener) IsEmulated() bool {
	return false
}

func (listener *SNlbListener) GetEgressMbps() int {
	return 0
}

func (listener *SNlbListener) Refresh() error {
	lis, err := listener.nlb.region.GetNlbListener(listener.ListenerId)
	if err != nil {
		return err
	}
	return jsonutils.Update(listener, lis)
}

func (listener *SNlbListener) GetListenerType() string {
	switch listener.ListenerProtocol {
	case "TCP":
		return api.LB_LISTENER_TYPE_TCP
	case "UDP":
		return api.LB_LISTENER_TYPE_UDP
	case "TCPSSL":
		return "TCPSSL"
	default:
		return listener.ListenerProtocol
	}
}

func (listener *SNlbListener) GetListenerPort() int {
	return listener.ListenerPort
}

func (listener *SNlbListener) GetBackendGroupId() string {
	return listener.ServerGroupId
}

func (listener *SNlbListener) GetBackendServerPort() int {
	return 0
}

func (listener *SNlbListener) GetScheduler() string {
	return ""
}

func (listener *SNlbListener) GetAclStatus() string {
	return ""
}

func (listener *SNlbListener) GetAclType() string {
	return ""
}

func (listener *SNlbListener) GetAclId() string {
	return ""
}

func (listener *SNlbListener) GetHealthCheck() string {
	return ""
}

func (listener *SNlbListener) GetHealthCheckType() string {
	return ""
}

func (listener *SNlbListener) GetHealthCheckDomain() string {
	return ""
}

func (listener *SNlbListener) GetHealthCheckURI() string {
	return ""
}

func (listener *SNlbListener) GetHealthCheckCode() string {
	return ""
}

func (listener *SNlbListener) GetHealthCheckRise() int {
	return 0
}

func (listener *SNlbListener) GetHealthCheckFail() int {
	return 0
}

func (listener *SNlbListener) GetHealthCheckTimeout() int {
	return 0
}

func (listener *SNlbListener) GetHealthCheckInterval() int {
	return 0
}

func (listener *SNlbListener) GetHealthCheckReq() string {
	return ""
}

func (listener *SNlbListener) GetHealthCheckExp() string {
	return ""
}

func (listener *SNlbListener) GetStickySession() string {
	return ""
}

func (listener *SNlbListener) GetStickySessionType() string {
	return ""
}

func (listener *SNlbListener) GetStickySessionCookie() string {
	return ""
}

func (listener *SNlbListener) GetStickySessionCookieTimeout() int {
	return 0
}

func (listener *SNlbListener) XForwardedForEnabled() bool {
	return false
}

func (listener *SNlbListener) GzipEnabled() bool {
	return false
}

func (listener *SNlbListener) GetCertificateId() string {
	if len(listener.CertificateIds) > 0 {
		return listener.CertificateIds[0]
	}
	return ""
}

func (listener *SNlbListener) GetTLSCipherPolicy() string {
	return listener.SecurityPolicyId
}

func (listener *SNlbListener) HTTP2Enabled() bool {
	return false
}

func (listener *SNlbListener) ChangeCertificate(ctx context.Context, opts *cloudprovider.ListenerCertificateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (listener *SNlbListener) SetAcl(ctx context.Context, opts *cloudprovider.ListenerAclOptions) error {
	return cloudprovider.ErrNotSupported
}

func (listener *SNlbListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return []cloudprovider.ICloudLoadbalancerListenerRule{}, nil
}

func (listener *SNlbListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (listener *SNlbListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	return nil, cloudprovider.ErrNotFound
}

func (listener *SNlbListener) Delete(ctx context.Context) error {
	return listener.nlb.region.DeleteNlbListener(listener.ListenerId)
}

func (listener *SNlbListener) Start() error {
	return listener.nlb.region.StartNlbListener(listener.ListenerId)
}

func (listener *SNlbListener) Stop() error {
	return listener.nlb.region.StopNlbListener(listener.ListenerId)
}

func (listener *SNlbListener) ChangeScheduler(ctx context.Context, opts *cloudprovider.ChangeListenerSchedulerOptions) error {
	return cloudprovider.ErrNotSupported
}

func (listener *SNlbListener) SetHealthCheck(ctx context.Context, opts *cloudprovider.ListenerHealthCheckOptions) error {
	return cloudprovider.ErrNotSupported
}

func (listener *SNlbListener) GetProjectId() string {
	return listener.nlb.GetProjectId()
}

func (listener *SNlbListener) GetClientIdleTimeout() int {
	return listener.IdleTimeout
}

func (listener *SNlbListener) GetBackendConnectTimeout() int {
	return 0
}

// region methods
func (region *SRegion) GetNlbListeners(loadBalancerId string) ([]SNlbListener, error) {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"MaxResults": "100",
	}

	if len(loadBalancerId) > 0 {
		params["LoadBalancerIds.1"] = loadBalancerId
	}

	listeners := []SNlbListener{}
	nextToken := ""

	for {
		if nextToken != "" {
			params["NextToken"] = nextToken
		}

		body, err := region.nlbRequest("ListListeners", params)
		if err != nil {
			return nil, err
		}

		pageListeners := []SNlbListener{}
		err = body.Unmarshal(&pageListeners, "Listeners")
		if err != nil {
			return nil, err
		}

		listeners = append(listeners, pageListeners...)

		nextToken, _ = body.GetString("NextToken")
		if nextToken == "" {
			break
		}
	}

	return listeners, nil
}

func (region *SRegion) GetNlbListener(listenerId string) (*SNlbListener, error) {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"ListenerId": listenerId,
	}

	body, err := region.nlbRequest("GetListenerAttribute", params)
	if err != nil {
		return nil, err
	}

	listener := &SNlbListener{}
	err = body.Unmarshal(listener)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (region *SRegion) CreateNlbListener(nlb *SNlb, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (*SNlbListener, error) {
	params := map[string]string{
		"RegionId":         region.RegionId,
		"LoadBalancerId":   nlb.LoadBalancerId,
		"ListenerProtocol": listener.ListenerType,
		"ListenerPort":     fmt.Sprintf("%d", listener.ListenerPort),
	}

	if len(listener.Name) > 0 {
		params["ListenerDescription"] = listener.Name
	}

	body, err := region.nlbRequest("CreateListener", params)
	if err != nil {
		return nil, err
	}

	listenerId, err := body.GetString("ListenerId")
	if err != nil {
		return nil, err
	}

	return region.GetNlbListener(listenerId)
}

func (region *SRegion) DeleteNlbListener(listenerId string) error {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"ListenerId": listenerId,
	}

	_, err := region.nlbRequest("DeleteListener", params)
	return err
}

func (region *SRegion) StartNlbListener(listenerId string) error {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"ListenerId": listenerId,
	}

	_, err := region.nlbRequest("StartListener", params)
	return err
}

func (region *SRegion) StopNlbListener(listenerId string) error {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"ListenerId": listenerId,
	}

	_, err := region.nlbRequest("StopListener", params)
	return err
}
