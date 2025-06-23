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

type SAlbListener struct {
	multicloud.SResourceBase
	multicloud.SLoadbalancerRedirectBase
	AliyunTags
	alb *SAlb

	ListenerId             string                 `json:"ListenerId"`
	ListenerDescription    string                 `json:"ListenerDescription"`
	ListenerProtocol       string                 `json:"ListenerProtocol"`
	ListenerPort           int                    `json:"ListenerPort"`
	ListenerStatus         string                 `json:"ListenerStatus"`
	LoadBalancerId         string                 `json:"LoadBalancerId"`
	RequestTimeout         int                    `json:"RequestTimeout"`
	IdleTimeout            int                    `json:"IdleTimeout"`
	SecurityPolicyId       string                 `json:"SecurityPolicyId"`
	GzipEnabledField       bool                   `json:"GzipEnabled"`
	Http2Enabled           bool                   `json:"Http2Enabled"`
	QuicEnabled            bool                   `json:"QuicEnabled"`
	ProxyProtocolEnabled   bool                   `json:"ProxyProtocolEnabled"`
	Certificates           []AlbCertificate       `json:"Certificates"`
	CaCertificates         []AlbCertificate       `json:"CaCertificates"`
	DefaultActions         []AlbAction            `json:"DefaultActions"`
	XForwardedForConfig    map[string]interface{} `json:"XForwardedForConfig"`
	AccessLogTracingConfig map[string]interface{} `json:"AccessLogTracingConfig"`
	// LogConfig              map[string]interface{} `json:"LogConfig"`
	CreateTime string `json:"CreateTime"`
	RegionId   string `json:"RegionId"`
}

type AlbCertificate struct {
	CertificateId string `json:"CertificateId"`
	IsDefault     bool   `json:"IsDefault"`
}

type AlbAction struct {
	Type                string              `json:"Type"`
	Order               int                 `json:"Order"`
	ForwardGroupConfig  ForwardGroupConfig  `json:"ForwardGroupConfig"`
	FixedResponseConfig FixedResponseConfig `json:"FixedResponseConfig"`
	RedirectConfig      RedirectConfig      `json:"RedirectConfig"`
	RewriteConfig       RewriteConfig       `json:"RewriteConfig"`
	InsertHeaderConfig  InsertHeaderConfig  `json:"InsertHeaderConfig"`
	RemoveHeaderConfig  RemoveHeaderConfig  `json:"RemoveHeaderConfig"`
	TrafficLimitConfig  TrafficLimitConfig  `json:"TrafficLimitConfig"`
	TrafficMirrorConfig TrafficMirrorConfig `json:"TrafficMirrorConfig"`
	CorsConfig          CorsConfig          `json:"CorsConfig"`
}

type ForwardGroupConfig struct {
	ServerGroupTuples []ServerGroupTuple `json:"ServerGroupTuples"`
}

type ServerGroupTuple struct {
	ServerGroupId string `json:"ServerGroupId"`
	Weight        int    `json:"Weight"`
}

type FixedResponseConfig struct {
	Content     string `json:"Content"`
	ContentType string `json:"ContentType"`
	HttpCode    string `json:"HttpCode"`
}

type RedirectConfig struct {
	Host     string `json:"Host"`
	HttpCode string `json:"HttpCode"`
	Path     string `json:"Path"`
	Port     string `json:"Port"`
	Protocol string `json:"Protocol"`
	Query    string `json:"Query"`
}

type RewriteConfig struct {
	Host  string `json:"Host"`
	Path  string `json:"Path"`
	Query string `json:"Query"`
}

type InsertHeaderConfig struct {
	Key       string `json:"Key"`
	Value     string `json:"Value"`
	ValueType string `json:"ValueType"`
}

type RemoveHeaderConfig struct {
	Key string `json:"Key"`
}

type TrafficLimitConfig struct {
	QPS int `json:"QPS"`
}

type TrafficMirrorConfig struct {
	MirrorGroupConfig MirrorGroupConfig `json:"MirrorGroupConfig"`
	TargetType        string            `json:"TargetType"`
}

type MirrorGroupConfig struct {
	ServerGroupTuples []ServerGroupTuple `json:"ServerGroupTuples"`
}

type CorsConfig struct {
	AllowCredentials string   `json:"AllowCredentials"`
	AllowHeaders     []string `json:"AllowHeaders"`
	AllowMethods     []string `json:"AllowMethods"`
	AllowOrigin      []string `json:"AllowOrigin"`
	ExposeHeaders    []string `json:"ExposeHeaders"`
	MaxAge           int      `json:"MaxAge"`
}

func (listener *SAlbListener) GetName() string {
	if len(listener.ListenerDescription) > 0 {
		return listener.ListenerDescription
	}
	return fmt.Sprintf("%s:%d", listener.ListenerProtocol, listener.ListenerPort)
}

func (listener *SAlbListener) GetId() string {
	return listener.ListenerId
}

func (listener *SAlbListener) GetGlobalId() string {
	return listener.ListenerId
}

func (listener *SAlbListener) GetStatus() string {
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

func (listener *SAlbListener) IsEmulated() bool {
	return false
}

func (listener *SAlbListener) GetEgressMbps() int {
	return 0
}

func (listener *SAlbListener) Refresh() error {
	lis, err := listener.alb.region.GetAlbListener(listener.ListenerId)
	if err != nil {
		return err
	}
	return jsonutils.Update(listener, lis)
}

func (listener *SAlbListener) GetListenerType() string {
	switch listener.ListenerProtocol {
	case "HTTP":
		return api.LB_LISTENER_TYPE_HTTP
	case "HTTPS":
		return api.LB_LISTENER_TYPE_HTTPS
	case "QUIC":
		return "QUIC"
	default:
		return listener.ListenerProtocol
	}
}

func (listener *SAlbListener) GetListenerPort() int {
	return listener.ListenerPort
}

func (listener *SAlbListener) GetBackendGroupId() string {
	if len(listener.DefaultActions) > 0 {
		for _, action := range listener.DefaultActions {
			if action.Type == "ForwardGroup" && len(action.ForwardGroupConfig.ServerGroupTuples) > 0 {
				return action.ForwardGroupConfig.ServerGroupTuples[0].ServerGroupId
			}
		}
	}
	return ""
}

func (listener *SAlbListener) GetBackendServerPort() int {
	return 0
}

func (listener *SAlbListener) GetScheduler() string {
	return ""
}

func (listener *SAlbListener) GetAclStatus() string {
	return ""
}

func (listener *SAlbListener) GetAclType() string {
	return ""
}

func (listener *SAlbListener) GetAclId() string {
	return ""
}

func (listener *SAlbListener) GetHealthCheck() string {
	return ""
}

func (listener *SAlbListener) GetHealthCheckType() string {
	return ""
}

func (listener *SAlbListener) GetHealthCheckDomain() string {
	return ""
}

func (listener *SAlbListener) GetHealthCheckURI() string {
	return ""
}

func (listener *SAlbListener) GetHealthCheckCode() string {
	return ""
}

func (listener *SAlbListener) GetHealthCheckRise() int {
	return 0
}

func (listener *SAlbListener) GetHealthCheckFail() int {
	return 0
}

func (listener *SAlbListener) GetHealthCheckTimeout() int {
	return 0
}

func (listener *SAlbListener) GetHealthCheckInterval() int {
	return 0
}

func (listener *SAlbListener) GetHealthCheckReq() string {
	return ""
}

func (listener *SAlbListener) GetHealthCheckExp() string {
	return ""
}

func (listener *SAlbListener) GetStickySession() string {
	return ""
}

func (listener *SAlbListener) GetStickySessionType() string {
	return ""
}

func (listener *SAlbListener) GetStickySessionCookie() string {
	return ""
}

func (listener *SAlbListener) GetStickySessionCookieTimeout() int {
	return 0
}

func (listener *SAlbListener) XForwardedForEnabled() bool {
	return false
}

func (listener *SAlbListener) GzipEnabled() bool {
	return listener.GzipEnabledField
}

func (listener *SAlbListener) GetCertificateId() string {
	for _, cert := range listener.Certificates {
		if cert.IsDefault {
			return cert.CertificateId
		}
	}
	if len(listener.Certificates) > 0 {
		return listener.Certificates[0].CertificateId
	}
	return ""
}

func (listener *SAlbListener) GetTLSCipherPolicy() string {
	return listener.SecurityPolicyId
}

func (listener *SAlbListener) HTTP2Enabled() bool {
	return listener.Http2Enabled
}

func (listener *SAlbListener) ChangeCertificate(ctx context.Context, opts *cloudprovider.ListenerCertificateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (listener *SAlbListener) SetAcl(ctx context.Context, opts *cloudprovider.ListenerAclOptions) error {
	return cloudprovider.ErrNotSupported
}

func (listener *SAlbListener) GetILoadbalancerListenerRules() ([]cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rules, err := listener.alb.region.GetAlbRules(listener.ListenerId)
	if err != nil {
		return nil, err
	}
	iRules := []cloudprovider.ICloudLoadbalancerListenerRule{}
	for i := 0; i < len(rules); i++ {
		rules[i].albListener = listener
		iRules = append(iRules, &rules[i])
	}
	return iRules, nil
}

func (listener *SAlbListener) CreateILoadBalancerListenerRule(rule *cloudprovider.SLoadbalancerListenerRule) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	listenerRule, err := listener.alb.region.CreateAlbRule(listener.ListenerId, rule)
	if err != nil {
		return nil, err
	}
	listenerRule.albListener = listener
	return listenerRule, nil
}

func (listener *SAlbListener) GetILoadBalancerListenerRuleById(ruleId string) (cloudprovider.ICloudLoadbalancerListenerRule, error) {
	rule, err := listener.alb.region.GetAlbRule(ruleId)
	if err != nil {
		return nil, err
	}
	rule.albListener = listener
	return rule, nil
}

func (listener *SAlbListener) Delete(ctx context.Context) error {
	return listener.alb.region.DeleteAlbListener(listener.ListenerId)
}

func (listener *SAlbListener) Start() error {
	return listener.alb.region.StartAlbListener(listener.ListenerId)
}

func (listener *SAlbListener) Stop() error {
	return listener.alb.region.StopAlbListener(listener.ListenerId)
}

func (listener *SAlbListener) ChangeScheduler(ctx context.Context, opts *cloudprovider.ChangeListenerSchedulerOptions) error {
	return cloudprovider.ErrNotSupported
}

func (listener *SAlbListener) SetHealthCheck(ctx context.Context, opts *cloudprovider.ListenerHealthCheckOptions) error {
	return cloudprovider.ErrNotSupported
}

func (listener *SAlbListener) GetProjectId() string {
	return listener.alb.GetProjectId()
}

func (listener *SAlbListener) GetClientIdleTimeout() int {
	return listener.IdleTimeout
}

func (listener *SAlbListener) GetBackendConnectTimeout() int {
	return listener.RequestTimeout
}

// region methods
func (region *SRegion) GetAlbListeners(loadBalancerId string) ([]SAlbListener, error) {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"MaxResults": "100",
	}

	if len(loadBalancerId) > 0 {
		params["LoadBalancerIds.1"] = loadBalancerId
	}

	listeners := []SAlbListener{}
	nextToken := ""

	for {
		if nextToken != "" {
			params["NextToken"] = nextToken
		}

		body, err := region.albRequest("ListListeners", params)
		if err != nil {
			return nil, err
		}

		pageListeners := []SAlbListener{}
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

func (region *SRegion) GetAlbListener(listenerId string) (*SAlbListener, error) {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"ListenerId": listenerId,
	}

	body, err := region.albRequest("GetListenerAttribute", params)
	if err != nil {
		return nil, err
	}

	listener := &SAlbListener{}
	err = body.Unmarshal(listener)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (region *SRegion) CreateAlbListener(alb *SAlb, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (*SAlbListener, error) {
	params := map[string]string{
		"RegionId":         region.RegionId,
		"LoadBalancerId":   alb.LoadBalancerId,
		"ListenerProtocol": listener.ListenerType,
		"ListenerPort":     fmt.Sprintf("%d", listener.ListenerPort),
	}

	if len(listener.Name) > 0 {
		params["ListenerDescription"] = listener.Name
	}

	body, err := region.albRequest("CreateListener", params)
	if err != nil {
		return nil, err
	}

	listenerId, err := body.GetString("ListenerId")
	if err != nil {
		return nil, err
	}

	return region.GetAlbListener(listenerId)
}

func (region *SRegion) DeleteAlbListener(listenerId string) error {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"ListenerId": listenerId,
	}

	_, err := region.albRequest("DeleteListener", params)
	return err
}

func (region *SRegion) StartAlbListener(listenerId string) error {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"ListenerId": listenerId,
	}

	_, err := region.albRequest("StartListener", params)
	return err
}

func (region *SRegion) StopAlbListener(listenerId string) error {
	params := map[string]string{
		"RegionId":   region.RegionId,
		"ListenerId": listenerId,
	}

	_, err := region.albRequest("StopListener", params)
	return err
}
