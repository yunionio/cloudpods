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

package huawei

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var LB_ALGORITHM_MAP = map[string]string{
	api.LB_SCHEDULER_WRR: "ROUND_ROBIN",
	api.LB_SCHEDULER_WLC: "LEAST_CONNECTIONS",
	api.LB_SCHEDULER_SCH: "SOURCE_IP",
}

var LB_PROTOCOL_MAP = map[string]string{
	api.LB_LISTENER_TYPE_HTTP:  "HTTP",
	api.LB_LISTENER_TYPE_HTTPS: "TERMINATED_HTTPS",
	api.LB_LISTENER_TYPE_UDP:   "UDP",
	api.LB_LISTENER_TYPE_TCP:   "TCP",
}

var LBBG_PROTOCOL_MAP = map[string]string{
	api.LB_LISTENER_TYPE_HTTP:  "HTTP",
	api.LB_LISTENER_TYPE_HTTPS: "HTTP",
	api.LB_LISTENER_TYPE_UDP:   "UDP",
	api.LB_LISTENER_TYPE_TCP:   "TCP",
}

var LB_STICKY_SESSION_MAP = map[string]string{
	api.LB_STICKY_SESSION_TYPE_INSERT: "HTTP_COOKIE",
	api.LB_STICKY_SESSION_TYPE_SERVER: "APP_COOKIE",
}

var LB_HEALTHCHECK_TYPE_MAP = map[string]string{
	api.LB_HEALTH_CHECK_HTTP: "HTTP",
	api.LB_HEALTH_CHECK_TCP:  "TCP",
	api.LB_HEALTH_CHECK_UDP:  "UDP_CONNECT",
}

type SLoadbalancer struct {
	region *SRegion
	subnet *SNetwork
	eip    *SEipAddress

	Description        string     `json:"description"`
	ProvisioningStatus string     `json:"provisioning_status"`
	TenantID           string     `json:"tenant_id"`
	ProjectID          string     `json:"project_id"`
	AdminStateUp       bool       `json:"admin_state_up"`
	Provider           string     `json:"provider"`
	Pools              []Pool     `json:"pools"`
	Listeners          []Listener `json:"listeners"`
	VipPortID          string     `json:"vip_port_id"`
	OperatingStatus    string     `json:"operating_status"`
	VipAddress         string     `json:"vip_address"`
	VipSubnetID        string     `json:"vip_subnet_id"`
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type Listener struct {
	ID string `json:"id"`
}

type Pool struct {
	ID string `json:"id"`
}

func (self *SLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if self.GetEip() == nil {
		return nil, nil
	}

	return self.eip, nil
}

func (self *SLoadbalancer) GetId() string {
	return self.ID
}

func (self *SLoadbalancer) GetName() string {
	return self.Name
}

func (self *SLoadbalancer) GetGlobalId() string {
	return self.ID
}

func (self *SLoadbalancer) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancer) Refresh() error {
	lb, err := self.region.GetLoadbalancer(self.GetId())
	if err != nil {
		return err
	}

	return jsonutils.Update(self, lb)
}

func (self *SLoadbalancer) IsEmulated() bool {
	return false
}

func (self *SLoadbalancer) GetMetadata() *jsonutils.JSONDict {
	meta := jsonutils.NewDict()

	return meta
}

func (self *SLoadbalancer) GetProjectId() string {
	return self.ProjectID
}

func (self *SLoadbalancer) GetAddress() string {
	return self.VipAddress
}

// todo: api.LB_ADDR_TYPE_INTERNET?
func (self *SLoadbalancer) GetAddressType() string {
	return api.LB_ADDR_TYPE_INTRANET
}

func (self *SLoadbalancer) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_VPC
}

func (self *SLoadbalancer) GetNetworkIds() []string {
	return []string{self.VipSubnetID}
}

func (self *SLoadbalancer) GetNetwork() *SNetwork {
	if self.subnet == nil {
		net, err := self.region.getNetwork(self.VipSubnetID)
		if err == nil {
			self.subnet = net
		}
	}

	return self.subnet
}

func (self *SLoadbalancer) GetEip() *SEipAddress {
	if self.eip == nil {
		eips, _ := self.region.GetEips()
		for i := range eips {
			eip := &eips[i]
			if eip.PortId == self.VipPortID {
				self.eip = eip
			}
		}
	}

	return self.eip
}

func (self *SLoadbalancer) GetVpcId() string {
	net := self.GetNetwork()
	if net != nil {
		return net.VpcID
	}

	return ""
}

func (self *SLoadbalancer) GetZoneId() string {
	net := self.GetNetwork()
	if net != nil {
		return net.AvailabilityZone
	}

	return ""
}

func (self *SLoadbalancer) GetLoadbalancerSpec() string {
	return ""
}

func (self *SLoadbalancer) GetChargeType() string {
	eip := self.GetEip()
	if eip != nil {
		return eip.GetInternetChargeType()
	}

	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SLoadbalancer) GetEgressMbps() int {
	eip := self.GetEip()
	if eip != nil {
		return eip.GetBandwidth()
	}

	return 0
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0141008275.html
func (self *SLoadbalancer) Delete() error {
	return self.region.DeleteLoadBalancer(self.GetId())
}

func (self *SLoadbalancer) Start() error {
	return nil
}

func (self *SLoadbalancer) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	ret, err := self.region.GetLoadBalancerListeners(self.GetId())
	if err != nil {
		return nil, err
	}

	iret := make([]cloudprovider.ICloudLoadbalancerListener, 0)
	for i := range ret {
		listener := ret[i]
		listener.lb = self
		iret = append(iret, &listener)
	}

	return iret, nil
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	ret, err := self.region.GetLoadBalancerBackendGroups(self.GetId())
	if err != nil {
		return nil, err
	}

	iret := make([]cloudprovider.ICloudLoadbalancerBackendGroup, 0)
	for i := range ret {
		bg := ret[i]
		bg.lb = self
		iret = append(iret, &bg)
	}

	return iret, nil
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561549.html
func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	ret, err := self.region.CreateLoadBalancerBackendGroup(group)
	ret.lb = self
	return &ret, err
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561563.html
func (self *SLoadbalancer) CreateHealthCheck(backendGroupId string, healthcheck *cloudprovider.SLoadbalancerHealthCheck) error {
	_, err := self.region.CreateLoadBalancerHealthCheck(backendGroupId, healthcheck)
	return err
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561548.html
func (self *SLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	ret := &SElbBackendGroup{}
	err := DoGet(self.region.ecsClient.ElbBackendGroup.Get, groupId, nil, ret)
	if err != nil {
		return nil, err
	}

	ret.lb = self
	ret.region = self.region
	return ret, nil
}

func (self *SLoadbalancer) CreateILoadBalancerListener(listener *cloudprovider.SLoadbalancerListener) (cloudprovider.ICloudLoadbalancerListener, error) {
	ret, err := self.region.CreateLoadBalancerListener(listener)
	if err != nil {
		return nil, err
	}

	ret.lb = self
	return &ret, nil
}

func (self *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	ret := &SElbListener{}
	err := DoGet(self.region.ecsClient.ElbListeners.Get, listenerId, nil, ret)
	if err != nil {
		return nil, err
	}

	ret.lb = self
	return ret, nil
}

func (self *SRegion) GetLoadbalancer(lgId string) (SLoadbalancer, error) {
	elb := SLoadbalancer{}
	err := DoGet(self.ecsClient.Elb.Get, lgId, nil, &elb)
	if err != nil {
		return elb, err
	}

	return elb, nil
}

func (self *SRegion) DeleteLoadBalancer(elbId string) error {
	return DoDelete(self.ecsClient.Elb.Delete, elbId, nil, nil)
}

func (self *SRegion) GetLoadBalancerListeners(lbId string) ([]SElbListener, error) {
	params := map[string]string{}
	if len(lbId) > 0 {
		params["loadbalancer_id"] = lbId
	}

	ret := make([]SElbListener, 0)
	err := doListAll(self.ecsClient.ElbListeners.List, params, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (self *SRegion) CreateLoadBalancerListener(listener *cloudprovider.SLoadbalancerListener) (SElbListener, error) {
	params := jsonutils.NewDict()
	listenerObj := jsonutils.NewDict()
	listenerObj.Set("name", jsonutils.NewString(listener.Name))
	listenerObj.Set("description", jsonutils.NewString(listener.Description))
	listenerObj.Set("protocol", jsonutils.NewString(LB_PROTOCOL_MAP[listener.ListenerType]))
	listenerObj.Set("protocol_port", jsonutils.NewInt(int64(listener.ListenerPort)))
	listenerObj.Set("loadbalancer_id", jsonutils.NewString(listener.LoadbalancerID))
	listenerObj.Set("http2_enable", jsonutils.NewBool(listener.EnableHTTP2))
	if len(listener.BackendGroupID) > 0 {
		listenerObj.Set("default_pool_id", jsonutils.NewString(listener.BackendGroupID))
	}

	if listener.ListenerType == api.LB_LISTENER_TYPE_HTTPS {
		listenerObj.Set("default_tls_container_ref", jsonutils.NewString(listener.CertificateID))
	}

	if listener.XForwardedFor {
		insertObj := jsonutils.NewDict()
		insertObj.Set("X-Forwarded-ELB-IP", jsonutils.NewBool(listener.XForwardedFor))
		listenerObj.Set("insert_headers", insertObj)
	}
	params.Set("listener", listenerObj)
	ret := SElbListener{}
	err := DoCreate(self.ecsClient.ElbListeners.Create, params, &ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561547.html
func (self *SRegion) GetLoadBalancerBackendGroups(elbId string) ([]SElbBackendGroup, error) {
	params := map[string]string{}
	if len(elbId) > 0 {
		params["loadbalancer_id"] = elbId
	}

	ret := make([]SElbBackendGroup, 0)
	err := doListAll(self.ecsClient.ElbBackendGroup.List, params, &ret)
	if err != nil {
		return nil, err
	}

	for i := range ret {
		ret[i].region = self
	}

	return ret, nil
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561547.html
func (self *SRegion) CreateLoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (SElbBackendGroup, error) {
	ret := SElbBackendGroup{}
	var protocol, scheduler string
	if s, ok := LB_ALGORITHM_MAP[group.Scheduler]; !ok {
		return ret, fmt.Errorf("CreateILoadBalancerBackendGroup unsupported scheduler %s", group.Scheduler)
	} else {
		scheduler = s
	}

	if t, ok := LBBG_PROTOCOL_MAP[group.ListenType]; !ok {
		return ret, fmt.Errorf("CreateILoadBalancerBackendGroup unsupported listener type %s", group.ListenType)
	} else {
		protocol = t
	}

	params := jsonutils.NewDict()
	poolObj := jsonutils.NewDict()
	poolObj.Set("project_id", jsonutils.NewString(self.client.projectId))
	poolObj.Set("name", jsonutils.NewString(group.Name))
	poolObj.Set("protocol", jsonutils.NewString(protocol))
	poolObj.Set("lb_algorithm", jsonutils.NewString(scheduler))

	if len(group.ListenerID) > 0 {
		poolObj.Set("listener_id", jsonutils.NewString(group.ListenerID))
	} else if len(group.LoadbalancerID) > 0 {
		poolObj.Set("loadbalancer_id", jsonutils.NewString(group.LoadbalancerID))
	} else {
		return ret, fmt.Errorf("CreateLoadBalancerBackendGroup one of listener id / loadbalancer id must be specified")
	}

	if group.StickySession != nil {
		s := jsonutils.NewDict()
		timeout := int64(group.StickySession.StickySessionCookieTimeout / 60)
		if group.ListenType == api.LB_LISTENER_TYPE_UDP || group.ListenType == api.LB_LISTENER_TYPE_TCP {
			s.Set("type", jsonutils.NewString("SOURCE_IP"))
			if timeout > 0 {
				s.Set("persistence_timeout", jsonutils.NewInt(timeout))
			}
		} else {
			s.Set("type", jsonutils.NewString(LB_STICKY_SESSION_MAP[group.StickySession.StickySessionType]))
			if len(group.StickySession.StickySessionCookie) > 0 {
				s.Set("cookie_name", jsonutils.NewString(group.StickySession.StickySessionCookie))
			} else {
				if timeout > 0 {
					s.Set("persistence_timeout", jsonutils.NewInt(timeout))
				}
			}
		}

		poolObj.Set("session_persistence", s)
	}
	params.Set("pool", poolObj)
	err := DoCreate(self.ecsClient.ElbBackendGroup.Create, params, &ret)
	if err != nil {
		return ret, err
	}

	if group.HealthCheck != nil {
		_, err := self.CreateLoadBalancerHealthCheck(ret.GetId(), group.HealthCheck)
		if err != nil {
			return ret, err
		}
	}

	ret.region = self
	return ret, nil
}

func (self *SRegion) CreateLoadBalancerHealthCheck(backendGroupID string, healthCheck *cloudprovider.SLoadbalancerHealthCheck) (SElbHealthCheck, error) {
	params := jsonutils.NewDict()
	healthObj := jsonutils.NewDict()
	healthObj.Set("delay", jsonutils.NewInt(int64(healthCheck.HealthCheckInterval)))
	healthObj.Set("max_retries", jsonutils.NewInt(int64(healthCheck.HealthCheckRise)))
	healthObj.Set("pool_id", jsonutils.NewString(backendGroupID))
	healthObj.Set("timeout", jsonutils.NewInt(int64(healthCheck.HealthCheckTimeout)))
	healthObj.Set("type", jsonutils.NewString(LB_HEALTHCHECK_TYPE_MAP[healthCheck.HealthCheckType]))
	if healthCheck.HealthCheckType == api.LB_HEALTH_CHECK_HTTP {
		if len(healthCheck.HealthCheckDomain) > 0 {
			healthObj.Set("domain_name", jsonutils.NewString(healthCheck.HealthCheckDomain))
		}

		if len(healthCheck.HealthCheckURI) > 0 {
			healthObj.Set("url_path", jsonutils.NewString(healthCheck.HealthCheckURI))
		}

		if len(healthCheck.HealthCheckHttpCode) > 0 {
			healthObj.Set("expected_codes", jsonutils.NewString(ToHuaweiHealthCheckHttpCode(healthCheck.HealthCheckHttpCode)))
		}
	}
	params.Set("healthmonitor", healthObj)

	ret := SElbHealthCheck{}
	err := DoCreate(self.ecsClient.ElbHealthCheck.Create, params, &ret)
	if err != nil {
		return ret, err
	}

	ret.region = self
	return ret, nil
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561564.html
func (self *SRegion) UpdateLoadBalancerHealthCheck(healthCheckID string, healthCheck *cloudprovider.SLoadbalancerHealthCheck) (SElbHealthCheck, error) {
	params := jsonutils.NewDict()
	healthObj := jsonutils.NewDict()
	healthObj.Set("delay", jsonutils.NewInt(int64(healthCheck.HealthCheckInterval)))
	healthObj.Set("max_retries", jsonutils.NewInt(int64(healthCheck.HealthCheckRise)))
	healthObj.Set("timeout", jsonutils.NewInt(int64(healthCheck.HealthCheckTimeout)))
	if healthCheck.HealthCheckType == api.LB_HEALTH_CHECK_HTTP {
		if len(healthCheck.HealthCheckDomain) > 0 {
			healthObj.Set("domain_name", jsonutils.NewString(healthCheck.HealthCheckDomain))
		}

		if len(healthCheck.HealthCheckURI) > 0 {
			healthObj.Set("url_path", jsonutils.NewString(healthCheck.HealthCheckURI))
		}

		if len(healthCheck.HealthCheckHttpCode) > 0 {
			healthObj.Set("expected_codes", jsonutils.NewString(ToHuaweiHealthCheckHttpCode(healthCheck.HealthCheckHttpCode)))
		}
	}
	params.Set("healthmonitor", healthObj)

	ret := SElbHealthCheck{}
	err := DoUpdate(self.ecsClient.ElbHealthCheck.Update, healthCheckID, params, &ret)
	if err != nil {
		return ret, err
	}

	ret.region = self
	return ret, nil
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561565.html
func (self *SRegion) DeleteLoadbalancerHealthCheck(healthCheckID string) error {
	return DoDelete(self.ecsClient.ElbHealthCheck.Delete, healthCheckID, nil, nil)
}
