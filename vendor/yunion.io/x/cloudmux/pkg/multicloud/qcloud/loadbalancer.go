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

package qcloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

const (
	LB_ADDR_TYPE_INTERNAL = "INTERNAL"
	LB_ADDR_TYPE_OPEN     = "OPEN"
)

type LB_TYPE int64

const (
	LB_TYPE_CLASSIC     = LB_TYPE(0)
	LB_TYPE_APPLICATION = LB_TYPE(1)
)

// https://cloud.tencent.com/document/api/214/30694#LoadBalancer
type SLoadbalancer struct {
	multicloud.SLoadbalancerBase
	QcloudTags
	region *SRegion

	Status            int64     `json:"Status"` // 0：创建中，1：正常运行
	Domain            string    `json:"Domain"`
	VpcId             string    `json:"VpcId"`
	Log               string    `json:"Log"`
	ProjectId         int64     `json:"ProjectId"`
	Snat              bool      `json:"Snat"`
	LoadBalancerId    string    `json:"LoadBalancerId"`
	LoadBalancerVips  []string  `json:"LoadBalancerVips"`
	LoadBalancerType  string    `json:"LoadBalancerType"` // 负载均衡实例的网络类型： OPEN：公网属性， INTERNAL：内网属性。
	LoadBalancerName  string    `json:"LoadBalancerName"`
	Forward           LB_TYPE   `json:"Forward"` // 应用型负载均衡标识，1：应用型负载均衡，0：传统型的负载均衡。
	StatusTime        time.Time `json:"StatusTime"`
	OpenBGP           int64     `json:"OpenBgp"` // 高防 LB 的标识，1：高防负载均衡 0：非高防负载均衡。
	CreateTime        time.Time `json:"CreateTime"`
	Isolation         int64     `json:"Isolation"` // 0：表示未被隔离，1：表示被隔离。
	SubnetId          string    `json:"SubnetId"`
	BackupZoneSet     []ZoneSet `json:"BackupZoneSet"`
	MasterZone        ZoneSet   `json:"MasterZone"`
	NetworkAttributes struct {
		InternetChargeType      string
		InternetMaxBandwidthOut int
	}
}

type ZoneSet struct {
	Zone     string `json:"Zone"`
	ZoneId   int64  `json:"ZoneId"`
	ZoneName string `json:"ZoneName"`
}

func (self *SLoadbalancer) GetLoadbalancerSpec() string {
	return ""
}

func (self *SLoadbalancer) GetChargeType() string {
	if len(self.NetworkAttributes.InternetChargeType) > 0 && self.NetworkAttributes.InternetChargeType != "TRAFFIC_POSTPAId_BY_HOUR" {
		return api.LB_CHARGE_TYPE_BY_BANDWIDTH
	}
	return api.LB_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SLoadbalancer) GetEgressMbps() int {
	return self.NetworkAttributes.InternetMaxBandwidthOut
}

// https://cloud.tencent.com/document/product/214/30689
func (self *SLoadbalancer) Delete(ctx context.Context) error {
	_, err := self.region.DeleteLoadbalancer(self.GetId())
	if err != nil {
		return err
	}
	return cloudprovider.WaitDeleted(self, 5*time.Second, 60*time.Second)
}

// 腾讯云loadbalance不支持启用/禁用
func (self *SLoadbalancer) Start() error {
	return nil
}

func (self *SLoadbalancer) Stop() error {
	return cloudprovider.ErrNotSupported
}

// 腾讯云无后端服务器组
func (self *SLoadbalancer) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	groups, err := self.GetILoadBalancerBackendGroups()
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		if group.GetId() == groupId {
			return group, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func onecloudHealthCodeToQcloud(codes string) int {
	qcode := 0
	for i, code := range HTTP_CODES {
		if strings.Contains(codes, code) {
			// 按位或然后再赋值qcode
			qcode |= 1 << uint(i)
		}
	}

	return qcode
}

// https://cloud.tencent.com/document/product/214/30693
// todo:  1.限制比较多必须加参数校验 2.Onecloud 不支持双向证书可能存在兼容性问题
// 应用型负载均衡 传统型不支持设置SNI
func (self *SLoadbalancer) CreateILoadBalancerListener(ctx context.Context, opts *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	listenId, err := self.region.CreateLoadbalancerListener(self.LoadBalancerId, opts)
	if err != nil {
		return nil, err
	}
	var lblis cloudprovider.ICloudLoadbalancerListener
	err = cloudprovider.Wait(3*time.Second, 30*time.Second, func() (bool, error) {
		lblis, err = self.GetILoadBalancerListenerById(listenId)
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotFound {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "Wait.Listener.Created")
	}
	return lblis, nil
}

func (self *SLoadbalancer) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	ret, err := self.region.GetLoadbalancerListener(self.LoadBalancerId, listenerId)
	if err != nil {
		return nil, err
	}
	ret.lb = self
	return ret, nil
}

func (self *SLoadbalancer) GetId() string {
	return self.LoadBalancerId
}

func (self *SLoadbalancer) GetName() string {
	return self.LoadBalancerName
}

func (self *SLoadbalancer) GetGlobalId() string {
	return self.LoadBalancerId
}

func (self *SLoadbalancer) GetStatus() string {
	switch self.Status {
	case 0:
		return api.LB_STATUS_INIT
	case 1:
		return api.LB_STATUS_ENABLED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SLoadbalancer) Refresh() error {
	lb, err := self.region.GetLoadbalancer(self.GetId())
	if err != nil {
		return err
	}

	return jsonutils.Update(self, lb)
}

// 腾讯云当前不支持一个LB绑定多个ip，每个LB只支持绑定一个ip
func (self *SLoadbalancer) GetAddress() string {
	for _, addr := range self.LoadBalancerVips {
		return addr
	}
	return ""
}

func (self *SLoadbalancer) GetAddressType() string {
	switch self.LoadBalancerType {
	case LB_ADDR_TYPE_INTERNAL:
		return api.LB_ADDR_TYPE_INTRANET
	case LB_ADDR_TYPE_OPEN:
		return api.LB_ADDR_TYPE_INTERNET
	default:
		return ""
	}
}

func (self *SLoadbalancer) GetNetworkType() string {
	if len(self.VpcId) > 0 {
		return api.LB_NETWORK_TYPE_VPC
	}
	return api.LB_NETWORK_TYPE_CLASSIC
}

func (self *SLoadbalancer) GetNetworkIds() []string {
	if len(self.SubnetId) == 0 {
		return []string{}
	}

	return []string{self.SubnetId}
}

func (self *SLoadbalancer) GetVpcId() string {
	return self.VpcId
}

func (self *SLoadbalancer) GetZoneId() string {
	if len(self.MasterZone.Zone) > 0 {
		return self.MasterZone.Zone
	}
	if len(self.SubnetId) > 0 {
		net, _ := self.region.GetNetwork(self.SubnetId)
		if net != nil {
			return net.Zone
		}
	}
	return ""
}

func (self *SLoadbalancer) GetZone1Id() string {
	if len(self.BackupZoneSet) > 0 {
		return self.BackupZoneSet[0].Zone
	}
	return ""
}

func (self *SLoadbalancer) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	listeners, err := self.region.GetLoadbalancerListeners(self.LoadBalancerId, nil, "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudLoadbalancerListener{}
	for i := range listeners {
		listeners[i].lb = self
		ret = append(ret, &listeners[i])
	}
	return ret, nil
}

func (self *SLoadbalancer) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	listeners, err := self.region.GetLoadbalancerListeners(self.LoadBalancerId, nil, "")
	if err != nil {
		return nil, err
	}
	lbbgs := []SLBBackendGroup{}
	for i := range listeners {
		lbbgs = append(lbbgs, SLBBackendGroup{
			lb:       self,
			listener: &listeners[i],
		})
	}

	ret := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	for i := range lbbgs {
		ret = append(ret, &lbbgs[i])
	}

	return ret, nil
}

func (self *SLoadbalancer) GetIEIP() (cloudprovider.ICloudEIP, error) {
	if self.LoadBalancerType == "OPEN" && len(self.LoadBalancerVips) > 0 {
		return &SEipAddress{
			region:      self.region,
			AddressId:   self.LoadBalancerId,
			AddressIp:   self.LoadBalancerVips[0],
			AddressType: EIP_STATUS_BIND,
			InstanceId:  self.LoadBalancerId,
			CreatedTime: self.CreateTime,
		}, nil
	}
	return nil, nil
}

func (self *SRegion) GetLoadbalancers(ids []string, limit, offset int) ([]SLoadbalancer, int, error) {
	params := map[string]string{}
	for i, id := range ids {
		params[fmt.Sprintf("LoadBalancerIds.%d", i)] = id
	}

	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	resp, err := self.clbRequest("DescribeLoadBalancers", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeLoadBalancers")
	}

	ret := []SLoadbalancer{}
	err = resp.Unmarshal(&ret, "LoadBalancerSet")
	if err != nil {
		return nil, 0, err
	}

	total, _ := resp.Float("TotalCount")
	return ret, int(total), nil
}

func (self *SRegion) GetLoadbalancer(id string) (*SLoadbalancer, error) {
	lbs, _, err := self.GetLoadbalancers([]string{id}, 1, 0)
	if err != nil {
		return nil, err
	}
	for i := range lbs {
		if lbs[i].LoadBalancerId == id {
			lbs[i].region = self
			return &lbs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

/*
返回requstid 用于异步任务查询
https://cloud.tencent.com/document/product/214/30689
*/
func (self *SRegion) DeleteLoadbalancer(lbid string) (string, error) {
	params := map[string]string{"LoadBalancerIds.0": lbid}
	resp, err := self.clbRequest("DeleteLoadBalancer", params)
	if err != nil {
		return "", err
	}
	return resp.GetString("RequestId")
}

// https://cloud.tencent.com/document/product/214/30683
// 任务的当前状态。 0：成功，1：失败，2：进行中
func (self *SRegion) GetLBTaskStatus(requestId string) (string, error) {
	params := map[string]string{"TaskId": requestId}
	resp, err := self.clbRequest("DescribeTaskStatus", params)
	if err != nil {
		return "", err
	}

	status, err := resp.Get("Status")
	if err != nil {
		log.Debugf("WaitTaskSuccess failed %s: %s", err, resp.String())
		return "", err
	}

	_status, err := status.Float()
	return fmt.Sprintf("%1.f", _status), err
}

func (self *SRegion) WaitLBTaskSuccess(requestId string, interval time.Duration, timeout time.Duration) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		status, err := self.GetLBTaskStatus(requestId)
		if err != nil {
			return err
		}
		if status == "0" {
			return nil
		}

		if status == "1" {
			return fmt.Errorf("Task %s failed.", requestId)
		}

		time.Sleep(interval)
	}

	return cloudprovider.ErrTimeout
}

func (self *SLoadbalancer) GetProjectId() string {
	return strconv.Itoa(int(self.ProjectId))
}

func (self *SLoadbalancer) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags("clb", "clb", []string{self.LoadBalancerId}, tags, replace)
}

// https://cloud.tencent.com/document/api/214/30692
func (self *SRegion) CreateILoadBalancer(opts *cloudprovider.SLoadbalancerCreateOptions) (cloudprovider.ICloudLoadbalancer, error) {
	params := map[string]string{
		"LoadBalancerName": opts.Name,
		"VpcId":            opts.VpcId,
	}

	LoadBalancerType := "INTERNAL"
	if opts.AddressType == api.LB_ADDR_TYPE_INTERNET {
		LoadBalancerType = "OPEN"
		switch opts.ChargeType {
		case api.LB_CHARGE_TYPE_BY_BANDWIDTH:
			pkgs, _, err := self.GetBandwidthPackages([]string{}, 0, 50)
			if err != nil {
				return nil, errors.Wrapf(err, "GetBandwidthPackages")
			}
			bps := opts.EgressMbps
			if bps == 0 {
				bps = 200
			}
			if len(pkgs) > 0 {
				pkgId := pkgs[0].BandwidthPackageId
				for _, pkg := range pkgs {
					if len(pkg.ResourceSet) < 100 {
						pkgId = pkg.BandwidthPackageId
						break
					}
				}
				params["BandwidthPackageId"] = pkgId
				params["InternetAccessible.InternetChargeType"] = "BANDWIDTH_PACKAGE"
				params["InternetAccessible.InternetMaxBandwidthOut"] = fmt.Sprintf("%d", bps)
			} else {
				params["InternetAccessible.InternetChargeType"] = "BANDWIDTH_POSTPAID_BY_HOUR"
				params["InternetAccessible.InternetMaxBandwidthOut"] = fmt.Sprintf("%d", bps)
			}
		default:
			bps := opts.EgressMbps
			if bps == 0 {
				bps = 200
			}
			params["InternetAccessible.InternetChargeType"] = "TRAFFIC_POSTPAID_BY_HOUR"
			params["InternetAccessible.InternetMaxBandwidthOut"] = fmt.Sprintf("%d", bps)

		}
	}

	params["LoadBalancerType"] = LoadBalancerType

	if len(opts.ProjectId) > 0 {
		params["ProjectId"] = opts.ProjectId
	}

	if opts.AddressType != api.LB_ADDR_TYPE_INTERNET {
		params["SubnetId"] = opts.NetworkIds[0]
	} else {
		// 公网类型ELB可支持多可用区
		if len(opts.ZoneId) > 0 {
			if len(opts.SlaveZoneId) > 0 {
				params["MasterZoneId"] = opts.ZoneId
			} else {
				params["ZoneId"] = opts.ZoneId
			}
		}
	}
	i := 0
	for k, v := range opts.Tags {
		params[fmt.Sprintf("Tags.%d.TagKey", i)] = k
		params[fmt.Sprintf("Tags.%d.TagValue", i)] = v
		i++
	}

	resp, err := func() (jsonutils.JSONObject, error) {
		_resp, err := self.clbRequest("CreateLoadBalancer", params)
		if err != nil {
			// 兼容不支持指定zone的账号
			if e, ok := err.(*sdkerrors.TencentCloudSDKError); ok && e.Code == "InvalidParameterValue" {
				delete(params, "ZoneId")
				delete(params, "MasterZoneId")
				return self.clbRequest("CreateLoadBalancer", params)
			}
		}
		return _resp, err
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "CreateLoadBalancer")
	}

	ret := struct {
		RequestId       string
		LoadBalancerIds []string
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	if len(ret.RequestId) == 0 || len(ret.LoadBalancerIds) != 1 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, resp.String())
	}
	err = self.WaitLBTaskSuccess(ret.RequestId, 5*time.Second, time.Minute*1)
	if err != nil {
		return nil, errors.Wrapf(err, "WaitLBTaskSuccess")
	}
	return self.GetLoadbalancer(ret.LoadBalancerIds[0])
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	lbs := []SLoadbalancer{}
	for {
		part, total, err := self.GetLoadbalancers(nil, 100, len(lbs))
		if err != nil {
			return nil, err
		}
		lbs = append(lbs, part...)
		if len(lbs) >= total || len(part) == 0 {
			break
		}
	}
	ret := []cloudprovider.ICloudLoadbalancer{}
	for i := range lbs {
		lbs[i].region = self
		ret = append(ret, &lbs[i])
	}
	return ret, nil
}

func (self *SRegion) GetILoadBalancerById(id string) (cloudprovider.ICloudLoadbalancer, error) {
	lb, err := self.GetLoadbalancer(id)
	if err != nil {
		return nil, err
	}
	return lb, nil
}
