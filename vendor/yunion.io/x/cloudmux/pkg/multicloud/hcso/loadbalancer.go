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

package hcso

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei"
)

var LB_ALGORITHM_MAP = map[string]string{
	api.LB_SCHEDULER_WRR: "ROUND_ROBIN",
	api.LB_SCHEDULER_WLC: "LEAST_CONNECTIONS",
	api.LB_SCHEDULER_SCH: "SOURCE_IP",
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
	multicloud.SLoadbalancerBase
	huawei.HuaweiTags
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
	net := self.GetNetwork()
	if net != nil {
		return []string{net.GetId()}
	}

	return []string{}
}

func (self *SLoadbalancer) GetNetwork() *SNetwork {
	if self.subnet == nil {
		port, err := self.region.GetPort(self.VipPortID)
		if err == nil {
			net, err := self.region.getNetwork(port.NetworkID)
			if err == nil {
				self.subnet = net
			} else {
				log.Debugf("huawei.SLoadbalancer.getNetwork %s", err)
			}
		} else {
			log.Debugf("huawei.SLoadbalancer.GetPort %s", err)
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
		z, err := self.region.getZoneById(net.AvailabilityZone)
		if err != nil {
			log.Infof("getZoneById %s %s", net.AvailabilityZone, err)
			return ""
		}

		return z.GetGlobalId()
	}

	return ""
}

func (self *SLoadbalancer) GetZone1Id() string {
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
func (self *SLoadbalancer) Delete(ctx context.Context) error {
	for _, res := range self.Pools {
		backends, err := self.region.getLoadBalancerBackends(res.ID)
		if err != nil {
			return errors.Wrapf(err, "get backend group %s backends", res.ID)
		}
		for _, backend := range backends {
			err := self.region.RemoveLoadBalancerBackend(res.ID, backend.ID)
			if err != nil {
				return errors.Wrapf(err, "RemoveLoadBalancerBackend")
			}
		}
		pool, err := self.region.GetLoadBalancerBackendGroup(res.ID)
		if err != nil {
			return errors.Wrapf(err, "GetLoadBalancerBackendGroup")
		}
		if len(pool.HealthMonitorID) > 0 {
			err = self.region.DeleteLoadbalancerHealthCheck(pool.HealthMonitorID)
			if err != nil {
				return errors.Wrapf(err, "delete health check")
			}
		}
		err = self.region.DeleteLoadBalancerBackendGroup(res.ID)
		if err != nil {
			return errors.Wrapf(err, "delete backend group %s", res.ID)
		}
	}
	for _, lis := range self.Listeners {
		err := self.region.DeleteElbListener(lis.ID)
		if err != nil {
			return errors.Wrapf(err, "delete listener %s", lis.ID)
		}
	}
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
func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(opts *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	ret, err := self.region.CreateLoadBalancerBackendGroup(self.ID, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateLoadBalancerBackendGroup")
	}
	ret.lb = self
	return ret, err
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

func (self *SLoadbalancer) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	ret, err := self.region.CreateLoadBalancerListener(self.ID, listener)
	if err != nil {
		return nil, err
	}
	ret.lb = self
	return ret, nil
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

func (self *SRegion) CreateLoadBalancerListener(lbId string, opts *cloudprovider.SLoadbalancerListenerCreateOptions) (*SElbListener, error) {
	params := jsonutils.NewDict()
	listenerObj := jsonutils.NewDict()
	listenerObj.Set("name", jsonutils.NewString(opts.Name))
	listenerObj.Set("description", jsonutils.NewString(opts.Description))
	switch opts.ListenerType {
	case api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP, api.LB_LISTENER_TYPE_HTTP:
		opts.ListenerType = strings.ToUpper(opts.ListenerType)
	case api.LB_LISTENER_TYPE_HTTPS:
		opts.ListenerType = "TERMINATED_HTTPS"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "protocol %s", opts.ListenerType)
	}

	listenerObj.Set("protocol", jsonutils.NewString(opts.ListenerType))
	listenerObj.Set("protocol_port", jsonutils.NewInt(int64(opts.ListenerPort)))
	listenerObj.Set("loadbalancer_id", jsonutils.NewString(lbId))
	listenerObj.Set("http2_enable", jsonutils.NewBool(opts.EnableHTTP2))
	if len(opts.BackendGroupId) > 0 {
		listenerObj.Set("default_pool_id", jsonutils.NewString(opts.BackendGroupId))
	}

	if opts.ListenerType == api.LB_LISTENER_TYPE_HTTPS {
		listenerObj.Set("default_tls_container_ref", jsonutils.NewString(opts.CertificateId))
	}

	if opts.XForwardedFor {
		insertObj := jsonutils.NewDict()
		insertObj.Set("X-Forwarded-ELB-IP", jsonutils.NewBool(opts.XForwardedFor))
		listenerObj.Set("insert_headers", insertObj)
	}
	params.Set("listener", listenerObj)
	ret := &SElbListener{}
	err := DoCreate(self.ecsClient.ElbListeners.Create, params, ret)
	if err != nil {
		return nil, err
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
func (self *SRegion) CreateLoadBalancerBackendGroup(lbId string, opts *cloudprovider.SLoadbalancerBackendGroup) (*SElbBackendGroup, error) {
	params := jsonutils.NewDict()
	poolObj := jsonutils.NewDict()
	switch opts.Scheduler {
	case api.LB_SCHEDULER_WRR:
		opts.Scheduler = "ROUND_ROBIN"
	case api.LB_SCHEDULER_WLC:
		opts.Scheduler = "LEAST_CONNECTIONS"
	case api.LB_SCHEDULER_SCH:
		opts.Scheduler = "SOURCE_IP"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "invalid scheduler %s", opts.Scheduler)
	}
	switch opts.Protocol {
	case api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP:
		opts.Protocol = strings.ToUpper(opts.Protocol)
	case api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS:
		opts.Protocol = "HTTP"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "invalid protocol %s", opts.Protocol)
	}

	poolObj.Set("project_id", jsonutils.NewString(self.client.projectId))
	poolObj.Set("name", jsonutils.NewString(opts.Name))
	poolObj.Set("protocol", jsonutils.NewString(opts.Protocol))
	poolObj.Set("lb_algorithm", jsonutils.NewString(opts.Scheduler))
	poolObj.Set("loadbalancer_id", jsonutils.NewString(lbId))
	params.Set("pool", poolObj)

	ret := &SElbBackendGroup{region: self}
	err := DoCreate(self.ecsClient.ElbBackendGroup.Create, params, ret)
	if err != nil {
		return nil, err
	}
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

func (self *SLoadbalancer) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}
