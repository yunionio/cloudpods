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

// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http:// www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openstack

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

var LB_ALGORITHM_MAP = map[string]string{
	api.LB_SCHEDULER_RR:  "ROUND_ROBIN",
	api.LB_SCHEDULER_WRR: "ROUND_ROBIN",
	api.LB_SCHEDULER_WLC: "LEAST_CONNECTIONS",
	api.LB_SCHEDULER_SCH: "SOURCE_IP",
	api.LB_SCHEDULER_TCH: "SOURCE_IP_PORT",
}

var LB_PROTOCOL_MAP = map[string]string{
	api.LB_LISTENER_TYPE_HTTP:             "HTTP",
	api.LB_LISTENER_TYPE_HTTPS:            "HTTPS",
	api.LB_LISTENER_TYPE_TERMINATED_HTTPS: "TERMINATED_HTTPS",
	api.LB_LISTENER_TYPE_UDP:              "UDP",
	api.LB_LISTENER_TYPE_TCP:              "TCP",
}

var LB_STICKY_SESSION_MAP = map[string]string{
	api.LB_STICKY_SESSION_TYPE_INSERT: "HTTP_COOKIE",
	api.LB_STICKY_SESSION_TYPE_SERVER: "APP_COOKIE",
}

var LB_HEALTHCHECK_TYPE_MAP = map[string]string{
	api.LB_HEALTH_CHECK_HTTP:  "HTTP",
	api.LB_HEALTH_CHECK_HTTPS: "HTTPS",
	api.LB_HEALTH_CHECK_TCP:   "TCP",
	api.LB_HEALTH_CHECK_UDP:   "UDP_CONNECT",
}

type SLoadbalancerCreateParams struct {
	Description      string   `json:"description,omitempty"`
	AdminStateUp     bool     `json:"admin_state_up,omitempty"`
	ProjectID        string   `json:"project_id,omitempty"`
	VipNetworkId     string   `json:"vip_network_id,omitempty"`
	VipSubnetID      string   `json:"vip_subnet_id,omitempty"`
	VipAddress       string   `json:"vip_address,omitempty"`
	Provider         string   `json:"provider,omitempty"`
	Name             string   `json:"name,omitempty"`
	VipQosPolicyID   string   `json:"vip_qos_policy_id,omitempty"`
	AvailabilityZone string   `json:"availability_zone,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}

type SLoadbalancerID struct {
	ID string `json:"id"`
}

type SPoolID struct {
	ID string `json:"id"`
}

type SMemberID struct {
	ID string `json:"id"`
}

type SListenerID struct {
	ID string `json:"id"`
}

type SL7PolicieID struct {
	ID string `json:"id"`
}

type SL7RuleID struct {
	ID string `json:"id"`
}

type SLoadbalancer struct {
	multicloud.SLoadbalancerBase
	multicloud.OpenStackTags
	region *SRegion

	Description        string        `json:"description"`
	AdminStateUp       bool          `json:"admin_state_up"`
	ProjectID          string        `json:"project_id"`
	ProvisioningStatus string        `json:"provisioning_status"`
	FlavorID           string        `json:"flavor_id"`
	VipSubnetID        string        `json:"vip_subnet_id"`
	ListenerIds        []SListenerID `json:"listeners"`
	VipAddress         string        `json:"vip_address"`
	VipNetworkID       string        `json:"vip_network_id"`
	VipPortID          string        `json:"vip_port_id"`
	Provider           string        `json:"provider"`
	PoolIds            []SPoolID     `json:"pools"`
	CreatedAt          string        `json:"created_at"`
	UpdatedAt          string        `json:"updated_at"`
	ID                 string        `json:"id"`
	OperatingStatus    string        `json:"operating_status"`
	Name               string        `json:"name"`
	VipQosPolicyID     string        `json:"vip_qos_policy_id"`
	AvailabilityZone   string        `json:"availability_zone"`
	Tags               []string      `json:"tags"`
}

func waitLbResStatus(res cloudprovider.ICloudResource, interval time.Duration, timeout time.Duration) error {
	err := cloudprovider.WaitMultiStatus(res, []string{api.LB_STATUS_ENABLED, api.LB_STATUS_UNKNOWN}, interval, timeout)
	if err != nil {
		return errors.Wrap(err, "waitLbResStatus(res, interval, timeout)")
	}
	if res.GetStatus() == api.LB_STATUS_UNKNOWN {
		return errors.Wrap(fmt.Errorf("status error"), "check status")
	}
	return nil
}

func (lb *SLoadbalancer) GetName() string {
	return lb.Name
}

func (lb *SLoadbalancer) GetId() string {
	return lb.ID
}

func (lb *SLoadbalancer) GetGlobalId() string {
	return lb.ID
}

func (lb *SLoadbalancer) GetStatus() string {
	switch lb.ProvisioningStatus {
	case "ACTIVE":
		return api.LB_STATUS_ENABLED
	case "PENDING_CREATE":
		return api.LB_CREATING
	case "PENDING_UPDATE":
		return api.LB_SYNC_CONF
	case "PENDING_DELETE":
		return api.LB_STATUS_DELETING
	case "DELETED":
		return api.LB_STATUS_DELETED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (lb *SLoadbalancer) GetAddress() string {
	return lb.VipAddress
}

func (lb *SLoadbalancer) GetAddressType() string {
	eip, err := lb.GetIEIP()
	if err != nil {
		return api.LB_ADDR_TYPE_INTRANET
	}
	if eip == nil {
		return api.LB_ADDR_TYPE_INTRANET
	}
	return api.LB_ADDR_TYPE_INTERNET
}

func (lb *SLoadbalancer) GetNetworkType() string {
	network, err := lb.region.GetVpc(lb.VipNetworkID)
	if err != nil {
		log.Error(errors.Wrapf(err, "lb.region.GetNetwork(%s)", lb.VipNetworkID))
	}
	if network.NetworkType == "flat" || network.NetworkType == "vlan" {
		return api.LB_NETWORK_TYPE_CLASSIC
	}
	return api.LB_NETWORK_TYPE_VPC
}

func (lb *SLoadbalancer) GetNetworkIds() []string {
	return []string{lb.VipSubnetID}
}

func (lb *SLoadbalancer) GetZoneId() string {
	return lb.AvailabilityZone
}

func (self *SLoadbalancer) GetZone1Id() string {
	return ""
}

func (lb *SLoadbalancer) IsEmulated() bool {
	return false
}

func (lb *SLoadbalancer) GetVpcId() string {
	return lb.VipNetworkID
}

func (lb *SLoadbalancer) Refresh() error {
	loadbalancer, err := lb.region.GetLoadbalancerbyId(lb.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(lb, loadbalancer)
}

func (region *SRegion) GetLoadbalancers() ([]SLoadbalancer, error) {
	loadbalancers := []SLoadbalancer{}
	resource := "/v2/lbaas/loadbalancers"
	query := url.Values{}
	for {
		resp, err := region.lbList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "lbList")
		}
		part := struct {
			Loadbalancers      []SLoadbalancer
			LoadbalancersLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		loadbalancers = append(loadbalancers, part.Loadbalancers...)
		marker := part.LoadbalancersLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	for i := 0; i < len(loadbalancers); i++ {
		loadbalancers[i].region = region
	}
	return loadbalancers, nil
}

func (region *SRegion) GetLoadbalancerbyId(loadbalancerId string) (*SLoadbalancer, error) {
	// region.client.Debug(true)
	body, err := region.lbGet(fmt.Sprintf("/v2/lbaas/loadbalancers/%s", loadbalancerId))
	if err != nil {
		return nil, errors.Wrapf(err, `region.lbGet(/v2/lbaas/loadbalancers/%s)`, loadbalancerId)
	}
	loadbalancer := SLoadbalancer{}
	err = body.Unmarshal(&loadbalancer, "loadbalancer")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal(loadbalancer)")
	}
	loadbalancer.region = region
	return &loadbalancer, nil
}

func (region *SRegion) CreateLoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (*SLoadbalancer, error) {
	type CreateParams struct {
		Loadbalancer SLoadbalancerCreateParams `json:"loadbalancer"`
	}
	params := CreateParams{}
	params.Loadbalancer.AdminStateUp = true
	params.Loadbalancer.AvailabilityZone = loadbalancer.ZoneID
	params.Loadbalancer.Name = loadbalancer.Name
	params.Loadbalancer.ProjectID = loadbalancer.ProjectId
	params.Loadbalancer.VipSubnetID = loadbalancer.NetworkIDs[0]
	params.Loadbalancer.VipAddress = loadbalancer.Address

	body, err := region.lbPost("/v2/lbaas/loadbalancers", jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, `region.lbPost("/v2/lbaas/loadbalancers", jsonutils.Marshal(params))`)
	}
	sloadbalancer := SLoadbalancer{}
	err = body.Unmarshal(&sloadbalancer, "loadbalancer")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal(sloadbalancer, loadbalancer)")
	}
	sloadbalancer.region = region
	if len(loadbalancer.EipID) > 0 {
		err = region.AssociateEipWithPortId(sloadbalancer.VipPortID, loadbalancer.EipID)
		if err != nil {
			return nil, errors.Wrapf(err, "region.AssociateEipWithPortId(%s, %s)", sloadbalancer.VipPortID, loadbalancer.EipID)
		}
	}
	return &sloadbalancer, nil
}

func (region *SRegion) DeleteLoadbalancer(loadbalancerId string) error {
	_, err := region.lbDelete(fmt.Sprintf("/v2/lbaas/loadbalancers/%s?cascade=True", loadbalancerId))
	if err != nil {
		return errors.Wrapf(err, `region.lbDelete(/v2/lbaas/loadbalancers/%s?cascade=True)`, loadbalancerId)
	}
	return nil
}

func (lb *SLoadbalancer) Delete(ctx context.Context) error {
	return lb.region.DeleteLoadbalancer(lb.ID)
}

func (lb *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	ibackendgroups := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	for i := 0; i < len(lb.PoolIds); i++ {
		pool, err := lb.region.GetLoadbalancerPoolById(lb.PoolIds[i].ID)
		if err != nil {
			return nil, errors.Wrapf(err, "lb.region.GetLoadbalancerPoolById(%s)", lb.PoolIds[i].ID)
		}
		ibackendgroups = append(ibackendgroups, pool)
	}
	return ibackendgroups, nil
}

func (lb *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	// ensure lb status
	err := waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "waitLbResStatus(lb, api.LB_STATUS_ENABLED, 10*time.Second, 8*time.Minute)")
	}
	// create pool
	spool, err := lb.region.CreateLoadbalancerPool(group)
	if err != nil {
		return nil, errors.Wrap(err, "lb.region.CreateLoadbalancerPool")
	}
	// wait spool
	err = waitLbResStatus(spool, 10*time.Second, 8*time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "waitLbResStatus(spool,  10*time.Second, 8*time.Minute)")
	}
	// create healthmonitor
	if group.HealthCheck != nil {

		healthmonitor, err := lb.region.CreateLoadbalancerHealthmonitor(spool.ID, group.HealthCheck)
		if err != nil {
			return nil, errors.Wrapf(err, "region.CreateLoadbalancerHealthmonitor(%s, group.HealthCheck)", spool.ID)
		}
		spool.healthmonitor = healthmonitor
	}
	// wait health monitor
	if spool.healthmonitor != nil {
		err = waitLbResStatus(spool.healthmonitor, 10*time.Second, 8*time.Minute)
		if err != nil {
			return nil, errors.Wrap(err, "waitLbResStatus(spool.healthmonitor,  10*time.Second, 8*time.Minute)")
		}
	}
	return spool, nil
}

func (lb *SLoadbalancer) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListener) (cloudprovider.ICloudLoadbalancerListener, error) {
	// ensure lb status
	err := waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "waitLbResStatus(lb, api.LB_STATUS_ENABLED, 10*time.Second, 8*time.Minute)")
	}
	slistener, err := lb.region.CreateLoadbalancerListener(lb.ID, listener)
	if err != nil {
		return nil, errors.Wrapf(err, "lb.region.CreateLoadbalancerListener(%s, listener)", lb.ID)
	}
	return slistener, nil
}

func (lb *SLoadbalancer) GetLoadbalancerSpec() string {
	return lb.Description
}

func (lb *SLoadbalancer) GetChargeType() string {
	eip, err := lb.GetIEIP()
	if err != nil {
		log.Errorf("lb.GetIEIP(): %s", err)
	}
	if err != nil {
		return eip.GetInternetChargeType()
	}

	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (lb *SLoadbalancer) GetEgressMbps() int {
	return 0
}

func (lb *SLoadbalancer) GetILoadBalancerBackendGroupById(poolId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	err := lb.Refresh()
	if err != nil {
		return nil, errors.Wrap(err, "lb.Refresh()")
	}
	index := -1
	for i := 0; i < len(lb.PoolIds); i++ {
		if poolId == lb.PoolIds[i].ID {
			index = i
		}
	}
	if index < 0 {
		return nil, cloudprovider.ErrNotFound
	}
	spool, err := lb.region.GetLoadbalancerPoolById(poolId)
	if err != nil {
		return nil, errors.Wrapf(err, "lb.region.GetLoadbalancerPoolById(%s)", poolId)
	}
	if spool.GetStatus() == api.LB_STATUS_DELETING {
		return nil, cloudprovider.ErrNotFound
	}
	return spool, nil
}

func (lb *SLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	eips, err := lb.region.GetEips("")
	if err != nil {
		return nil, errors.Wrapf(err, "lb.region.GetEips()")
	}
	for _, eip := range eips {
		if eip.PortId == lb.VipPortID {
			return &eip, nil
		}
	}
	return nil, nil
}

func (region *SRegion) UpdateLoadBalancerAdminStateUp(AdminStateUp bool, loadbalancerId string) error {
	params := jsonutils.NewDict()
	poolParam := jsonutils.NewDict()
	poolParam.Add(jsonutils.NewBool(AdminStateUp), "admin_state_up")
	params.Add(poolParam, "loadbalancer")
	_, err := region.lbUpdate(fmt.Sprintf("/v2/lbaas/loadbalancers/%s", loadbalancerId), params)
	if err != nil {
		return errors.Wrapf(err, `region.lbUpdate(/v2/lbaas/loadbalancers/%s), params)`, loadbalancerId)
	}
	return nil
}

func (lb *SLoadbalancer) Start() error {
	// ensure lb status
	err := waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waitLbResStatus(lb, api.LB_STATUS_ENABLED, 10*time.Second, 8*time.Minute)")
	}
	err = lb.region.UpdateLoadBalancerAdminStateUp(true, lb.ID)
	if err != nil {
		return errors.Wrapf(err, "lb.region.UpdateLoadBalancerAdminStateUp(true, %s)", lb.ID)
	}
	err = waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waitLbResStatus(lb,  10*time.Second, 8*time.Minute)")
	}
	return nil
}

func (lb *SLoadbalancer) Stop() error {
	// ensure lb status
	err := waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waitLbResStatus(lb, api.LB_STATUS_ENABLED, 10*time.Second, 8*time.Minute)")
	}
	err = lb.region.UpdateLoadBalancerAdminStateUp(false, lb.ID)
	if err != nil {
		return errors.Wrapf(err, "lb.region.UpdateLoadBalancerAdminStateUp(false,%s)", lb.ID)
	}
	err = waitLbResStatus(lb, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waitLbResStatus(lb,  10*time.Second, 8*time.Minute)")
	}
	return nil
}

func (lb *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {

	return lb.region.GetLoadbalancerListenerbyId(listenerId)
}

func (lb *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	ilisteners := []cloudprovider.ICloudLoadbalancerListener{}
	for i := 0; i < len(lb.ListenerIds); i++ {
		listener, err := lb.region.GetLoadbalancerListenerbyId(lb.ListenerIds[i].ID)
		if err != nil {
			return nil, errors.Wrapf(err, "lb.region.GetLoadbalancerListenerbyId(%s)", lb.ListenerIds[i].ID)
		}
		ilisteners = append(ilisteners, listener)
	}
	return ilisteners, nil
}

func (lb *SLoadbalancer) GetProjectId() string {
	return lb.ProjectID
}

func (self *SLoadbalancer) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}
