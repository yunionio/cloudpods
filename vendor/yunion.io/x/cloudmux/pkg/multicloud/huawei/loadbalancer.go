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
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	multicloud.SResourceBase
	HuaweiTags
	region *SRegion
	subnet *SNetwork

	Description         string     `json:"description"`
	ProvisioningStatus  string     `json:"provisioning_status"`
	TenantId            string     `json:"tenant_id"`
	ProjectId           string     `json:"project_id"`
	EnterpriseProjectId string     `json:"enterprise_project_id"`
	AdminStateUp        bool       `json:"admin_state_up"`
	Provider            string     `json:"provider"`
	Pools               []Pool     `json:"pools"`
	Listeners           []Listener `json:"listeners"`
	VipPortId           string     `json:"vip_port_id"`
	OperatingStatus     string     `json:"operating_status"`
	VipAddress          string     `json:"vip_address"`
	VipSubnetId         string     `json:"vip_subnet_id"`
	Id                  string     `json:"id"`
	Name                string     `json:"name"`
	VpcId               string
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Publicips           []struct {
		PublicipId      string
		PublicipAddress string
		IpVersion       string
	}
}

type Listener struct {
	Id string `json:"id"`
}

type Pool struct {
	Id string `json:"id"`
}

func (self *SLoadbalancer) GetEip() (*SEipAddress, error) {
	for _, ip := range self.Publicips {
		eip, err := self.region.GetEip(ip.PublicipId)
		if err != nil {
			return nil, err
		}
		return eip, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	eip, err := self.GetEip()
	if err != nil {
		return nil, err
	}
	return eip, nil
}

func (self *SLoadbalancer) GetId() string {
	return self.Id
}

func (self *SLoadbalancer) GetName() string {
	return self.Name
}

func (self *SLoadbalancer) GetGlobalId() string {
	return self.Id
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

func (self *SLoadbalancer) GetProjectId() string {
	return self.EnterpriseProjectId
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
		port, err := self.region.GetPort(self.VipPortId)
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

func (self *SLoadbalancer) GetVpcId() string {
	return self.VpcId
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
	eip, _ := self.GetEip()
	if eip != nil {
		return eip.GetInternetChargeType()
	}
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SLoadbalancer) GetEgressMbps() int {
	eip, _ := self.GetEip()
	if eip != nil {
		return eip.GetBandwidth()
	}

	return 0
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0141008275.html
func (self *SLoadbalancer) Delete(ctx context.Context) error {
	for _, res := range self.Pools {
		backends, err := self.region.getLoadBalancerBackends(res.Id)
		if err != nil {
			return errors.Wrapf(err, "get backend group %s backends", res.Id)
		}
		for _, backend := range backends {
			err := self.region.RemoveLoadBalancerBackend(res.Id, backend.ID)
			if err != nil {
				return errors.Wrapf(err, "RemoveLoadBalancerBackend")
			}
		}
		pool, err := self.region.GetLoadBalancerBackendGroup(res.Id)
		if err != nil {
			return errors.Wrapf(err, "GetLoadBalancerBackendGroup")
		}
		if len(pool.HealthMonitorID) > 0 {
			err = self.region.DeleteLoadbalancerHealthCheck(pool.HealthMonitorID)
			if err != nil {
				return errors.Wrapf(err, "delete health check")
			}
		}
		err = self.region.DeleteLoadBalancerBackendGroup(res.Id)
		if err != nil {
			return errors.Wrapf(err, "delete backend group %s", res.Id)
		}
	}
	for _, lis := range self.Listeners {
		err := self.region.DeleteElbListener(lis.Id)
		if err != nil {
			return errors.Wrapf(err, "delete listener %s", lis.Id)
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
		bg.region = self.region
		iret = append(iret, &bg)
	}

	return iret, nil
}

func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(opts *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	ret, err := self.region.CreateLoadBalancerBackendGroup(self.Id, opts)
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
	ret := &SElbBackendGroup{lb: self, region: self.region}
	resp, err := self.region.list(SERVICE_ELB, "elb/pools/"+groupId, nil)
	if err != nil {
		return nil, err
	}
	return ret, resp.Unmarshal(ret, "pool")
}

func (self *SLoadbalancer) CreateILoadBalancerListener(ctx context.Context, opts *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	ret, err := self.region.CreateLoadBalancerListener(opts, self.Id)
	if err != nil {
		return nil, err
	}
	ret.lb = self
	return ret, nil
}

func (self *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	ret := &SElbListener{lb: self}
	resp, err := self.region.list(SERVICE_ELB, "elb/listeners/"+listenerId, nil)
	if err != nil {
		return nil, err
	}
	return ret, resp.Unmarshal(ret, "listener")
}

func (self *SRegion) GetLoadbalancer(id string) (*SLoadbalancer, error) {
	resp, err := self.list(SERVICE_ELB, "elb/loadbalancers/"+id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SLoadbalancer{region: self}
	return ret, resp.Unmarshal(ret, "loadbalancer")
}

func (self *SRegion) DeleteLoadBalancer(elbId string) error {
	resource := fmt.Sprintf("elb/loadbalancers/%s", elbId)
	_, err := self.delete(SERVICE_ELB, resource)
	return err
}

func (self *SRegion) GetLoadBalancerListeners(lbId string) ([]SElbListener, error) {
	ret := []SElbListener{}
	params := url.Values{}
	if len(lbId) > 0 {
		params.Set("loadbalancer_id", lbId)
	}
	return ret, self.lbListAll("elb/listeners", params, "listeners", &ret)
}

func (self *SRegion) CreateLoadBalancerListener(listener *cloudprovider.SLoadbalancerListenerCreateOptions, lbId string) (*SElbListener, error) {
	params := map[string]interface{}{
		"name":            listener.Name,
		"description":     listener.Description,
		"protocol_port":   listener.ListenerPort,
		"loadbalancer_id": lbId,
		"http2_enable":    listener.EnableHTTP2,
	}
	switch listener.ListenerType {
	case api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP, api.LB_LISTENER_TYPE_HTTP:
		params["protocol"] = strings.ToUpper(listener.ListenerType)
	case api.LB_LISTENER_TYPE_HTTPS:
		params["protocol"] = "TERMINATED_HTTPS"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "protocol %s", listener.ListenerType)
	}
	if len(listener.BackendGroupId) > 0 {
		params["default_pool_id"] = listener.BackendGroupId
	}

	if listener.ListenerType == api.LB_LISTENER_TYPE_HTTPS {
		params["default_tls_container_ref"] = listener.CertificateId
	}

	if listener.XForwardedFor {
		params["insert_headers"] = map[string]interface{}{
			"X-Forwarded-ELB-IP": listener.XForwardedFor,
		}
	}

	ret := &SElbListener{}
	resp, err := self.post(SERVICE_ELB, "elb/listeners", map[string]interface{}{"listener": params})
	if err != nil {
		return nil, err
	}
	return ret, resp.Unmarshal(&ret, "listener")
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561547.html
func (self *SRegion) GetLoadBalancerBackendGroups(elbId string) ([]SElbBackendGroup, error) {
	query := url.Values{}
	if len(elbId) > 0 {
		query.Set("loadbalancer_id", elbId)
	}

	ret := []SElbBackendGroup{}
	return ret, self.lbListAll("elb/pools", query, "pools", &ret)
}

func (self *SRegion) CreateLoadBalancerBackendGroup(lbId string, opts *cloudprovider.SLoadbalancerBackendGroup) (*SElbBackendGroup, error) {
	params := map[string]interface{}{
		"name":            opts.Name,
		"loadbalancer_id": lbId,
	}
	switch opts.Scheduler {
	case api.LB_SCHEDULER_WRR:
		params["lb_algorithm"] = "ROUND_ROBIN"
	case api.LB_SCHEDULER_WLC:
		params["lb_algorithm"] = "LEAST_CONNECTIONS"
	case api.LB_SCHEDULER_SCH:
		params["lb_algorithm"] = "SOURCE_IP"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "invalid scheduler %s", opts.Scheduler)
	}
	switch opts.Protocol {
	case api.LB_LISTENER_TYPE_TCP, api.LB_LISTENER_TYPE_UDP:
		params["protocol"] = strings.ToUpper(opts.Protocol)
	case api.LB_LISTENER_TYPE_HTTP, api.LB_LISTENER_TYPE_HTTPS:
		params["protocol"] = "HTTP"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "invalid protocol %s", opts.Protocol)
	}

	resp, err := self.post(SERVICE_ELB, "elb/pools", map[string]interface{}{"pool": params})
	if err != nil {
		return nil, err
	}
	ret := &SElbBackendGroup{region: self}
	err = resp.Unmarshal(ret, "pool")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) CreateLoadBalancerHealthCheck(backendGroupId string, healthCheck *cloudprovider.SLoadbalancerHealthCheck) (SElbHealthCheck, error) {
	params := map[string]interface{}{
		"delay":       healthCheck.HealthCheckInterval,
		"max_retries": healthCheck.HealthCheckRise,
		"pool_id":     backendGroupId,
		"timeout":     healthCheck.HealthCheckTimeout,
		"type":        LB_HEALTHCHECK_TYPE_MAP[healthCheck.HealthCheckType],
	}
	if healthCheck.HealthCheckType == api.LB_HEALTH_CHECK_HTTP {
		if len(healthCheck.HealthCheckDomain) > 0 {
			params["domain_name"] = healthCheck.HealthCheckDomain
		}

		if len(healthCheck.HealthCheckURI) > 0 {
			params["url_path"] = healthCheck.HealthCheckURI
		}

		if len(healthCheck.HealthCheckHttpCode) > 0 {
			params["expected_codes"] = ToHuaweiHealthCheckHttpCode(healthCheck.HealthCheckHttpCode)
		}
	}

	ret := SElbHealthCheck{region: self}
	resp, err := self.post(SERVICE_ELB, "elb/healthmonitors", map[string]interface{}{"healthmonitor": params})
	if err != nil {
		return ret, err
	}

	return ret, resp.Unmarshal(&ret, "healthmonitor")
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561564.html
func (self *SRegion) UpdateLoadBalancerHealthCheck(healthCheckId string, healthCheck *cloudprovider.SLoadbalancerHealthCheck) (SElbHealthCheck, error) {
	params := map[string]interface{}{
		"delay":       healthCheck.HealthCheckInterval,
		"max_retries": healthCheck.HealthCheckRise,
		"timeout":     healthCheck.HealthCheckTimeout,
	}
	if healthCheck.HealthCheckType == api.LB_HEALTH_CHECK_HTTP {
		if len(healthCheck.HealthCheckDomain) > 0 {
			params["domain_name"] = healthCheck.HealthCheckDomain
		}

		if len(healthCheck.HealthCheckURI) > 0 {
			params["url_path"] = healthCheck.HealthCheckURI
		}

		if len(healthCheck.HealthCheckHttpCode) > 0 {
			params["expected_codes"] = ToHuaweiHealthCheckHttpCode(healthCheck.HealthCheckHttpCode)
		}
	}

	ret := SElbHealthCheck{region: self}
	resp, err := self.put(SERVICE_ELB, "elb/healthmonitors/"+healthCheckId, map[string]interface{}{"healthmonitor": params})
	if err != nil {
		return ret, err
	}
	return ret, resp.Unmarshal(&ret, "healthmonitor")
}

// https://support.huaweicloud.com/api-elb/zh-cn_topic_0096561565.html
func (self *SRegion) DeleteLoadbalancerHealthCheck(healthCheckId string) error {
	_, err := self.delete(SERVICE_ELB, "elb/healthmonitors/"+healthCheckId)
	return err
}

func (self *SLoadbalancer) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) lbListAll(resource string, query url.Values, respKey string, retVal interface{}) error {
	ret := jsonutils.NewArray()
	for {
		resp, err := self.list(SERVICE_ELB, resource, query)
		if err != nil {
			return err
		}
		arr, err := resp.GetArray(respKey)
		if err != nil {
			return errors.Wrapf(err, "get %s", respKey)
		}
		ret.Add(arr...)
		marker, _ := resp.GetString("page_info", "next_marker")
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return ret.Unmarshal(retVal)
}

func (self *SRegion) CreateLoadBalancer(opts *cloudprovider.SLoadbalancerCreateOptions) (*SLoadbalancer, error) {
	subnet, err := self.getNetwork(opts.NetworkIds[0])
	if err != nil {
		return nil, errors.Wrap(err, "getNetwork")
	}

	params := map[string]interface{}{
		"name":               opts.Name,
		"description":        opts.Desc,
		"vip_subnet_cidr_id": subnet.NeutronSubnetID,
		"provider":           "vlb",
		"admin_state_up":     true,
		"guaranteed":         true,
		"project_id":         self.client.projectId,
		"charge_mode":        "lcu",
	}
	if len(opts.ProjectId) > 0 {
		params["enterprise_project_id"] = opts.ProjectId
	}
	if len(opts.Address) > 0 {
		params["vip_address"] = opts.Address
	}
	tags := []map[string]string{}
	for k, v := range opts.Tags {
		tags = append(tags, map[string]string{
			"key":   k,
			"value": v,
		})
	}
	if len(tags) > 0 {
		params["tags"] = tags
	}
	if len(opts.EipId) > 0 {
		params["publicip_ids"] = []string{opts.EipId}
	}
	err = self.fetchZones()
	if err != nil {
		return nil, errors.Wrapf(err, "fetchZones")
	}
	zoneIds := []string{}
	for i := range self.izones {
		zone := self.izones[i].(*SZone)
		zoneIds = append(zoneIds, zone.ZoneName)
	}
	params["availability_zone_list"] = zoneIds

	resp, err := self.post(SERVICE_ELB, "elb/loadbalancers", map[string]interface{}{"loadbalancer": params})
	if err != nil {
		return nil, err
	}
	ret := &SLoadbalancer{region: self}
	err = resp.Unmarshal(ret, "loadbalancer")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}
