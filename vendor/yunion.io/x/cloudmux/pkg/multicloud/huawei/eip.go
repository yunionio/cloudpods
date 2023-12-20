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
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SEipAddress struct {
	region *SRegion
	multicloud.SEipBase
	HuaweiTags

	Alias           string
	Id              string
	Status          string
	Type            string
	PublicIPAddress string
	CreateTime      time.Time
	Bandwidth       struct {
		Id         string
		Size       int
		ShareType  string
		ChargeMode string
		Name       string
	}
	BillingInfo           string
	EnterpriseProjectId   string
	AssociateInstanceType string
	AssociateInstanceId   string
	IPVersion             int64
	PortId                string
}

func (self *SEipAddress) GetId() string {
	return self.Id
}

func (self *SEipAddress) GetName() string {
	if len(self.Alias) > 0 {
		return self.Alias
	}
	return self.PublicIPAddress
}

func (self *SEipAddress) GetGlobalId() string {
	return self.Id
}

func (self *SEipAddress) GetStatus() string {
	switch self.Status {
	case "ACTIVE", "DOWN", "ELB":
		return api.EIP_STATUS_READY
	case "PENDING_CREATE", "NOTIFYING":
		return api.EIP_STATUS_ALLOCATE
	case "BINDING":
		return api.EIP_STATUS_ALLOCATE
	case "BIND_ERROR":
		return api.EIP_STATUS_ALLOCATE_FAIL
	case "PENDING_DELETE", "NOTIFY_DELETE":
		return api.EIP_STATUS_DEALLOCATE
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (self *SEipAddress) Refresh() error {
	eip, err := self.region.GetEip(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, eip)
}

func (self *SEipAddress) GetIpAddr() string {
	return self.PublicIPAddress
}

func (self *SEipAddress) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEipAddress) GetAssociationType() string {
	switch self.AssociateInstanceType {
	case "ELB", "ELBV1":
		return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
	case "NATGW":
		return api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
	case "PORT":
		return api.EIP_ASSOCIATE_TYPE_SERVER
	default:
		return strings.ToLower(self.AssociateInstanceType)
	}
}

func (self *SEipAddress) GetAssociationExternalId() string {
	if self.AssociateInstanceType == "PORT" {
		port, err := self.region.GetPort(self.AssociateInstanceId)
		if err == nil {
			return port.DeviceID
		}
	}
	return self.AssociateInstanceId
}

func (self *SEipAddress) GetBandwidth() int {
	return self.Bandwidth.Size
}

func (self *SEipAddress) GetInternetChargeType() string {
	if self.Bandwidth.ChargeMode == "traffic" {
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	}
	return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
}

func (self *SEipAddress) GetBillingType() string {
	if len(self.BillingInfo) > 0 {
		return billing_api.BILLING_TYPE_POSTPAID
	}
	return billing_api.BILLING_TYPE_PREPAID
}

func (self *SEipAddress) GetCreatedAt() time.Time {
	return self.CreateTime
}

func (self *SEipAddress) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SEipAddress) Delete() error {
	return self.region.DeallocateEIP(self.Id)
}

func (self *SEipAddress) Associate(opts *cloudprovider.AssociateConfig) error {
	switch opts.AssociateType {
	case api.EIP_ASSOCIATE_TYPE_SERVER:
		portId, err := self.region.GetInstancePortId(opts.InstanceId)
		if err != nil {
			return errors.Wrapf(err, "GetInstancePortId")
		}
		if len(self.PortId) > 0 {
			if self.PortId == portId {
				return nil
			}
			return fmt.Errorf("eip %s aready associate with port %s", self.GetId(), self.PortId)
		}
		err = self.region.AssociateEip(self.Id, portId, "PORT")
		if err != nil {
			return err
		}
	case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
		err := self.region.AssociateEip(self.Id, opts.InstanceId, "ELB")
		if err != nil {
			return err
		}
	case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
		err := self.region.AssociateEip(self.Id, opts.InstanceId, "NATGW")
		if err != nil {
			return err
		}
	default:
		return errors.Wrapf(cloudprovider.ErrNotSupported, "associate type %s", opts.AssociateType)
	}

	err := cloudprovider.WaitStatusWithDelay(self, api.EIP_STATUS_READY, 10*time.Second, 10*time.Second, 180*time.Second)
	return err
}

func (self *SEipAddress) Dissociate() error {
	err := self.region.DissociateEip(self.Id)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatus(self, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
}

func (self *SEipAddress) ChangeBandwidth(bw int) error {
	return self.region.UpdateEipBandwidth(self.Bandwidth.Id, bw)
}

func (self *SRegion) GetInstancePortId(instanceId string) (string, error) {
	// 目前只绑定一个网卡
	// todo: 还需要按照ports状态进行过滤
	ports, err := self.GetPorts(instanceId)
	if err != nil {
		return "", err
	}

	if len(ports) == 0 {
		return "", fmt.Errorf("AssociateEip instance %s port is empty", instanceId)
	}

	return ports[0].ID, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EIP/doc?version=v2&api=CreatePublicip
func (self *SRegion) AllocateEIP(opts *cloudprovider.SEip) (*SEipAddress, error) {
	if len(opts.BGPType) == 0 {
		switch self.GetId() {
		case "cn-north-1", "cn-east-2", "cn-south-1":
			opts.BGPType = "5_sbgp"
		case "cn-northeast-1":
			opts.BGPType = "5_telcom"
		case "cn-north-4", "ap-southeast-1", "ap-southeast-2", "eu-west-0":
			opts.BGPType = "5_bgp"
		case "cn-southwest-2":
			opts.BGPType = "5_sbgp"
		default:
			opts.BGPType = "5_bgp"
		}
	}

	// 华为云EIP名字最大长度64
	if len(opts.Name) > 64 {
		opts.Name = opts.Name[:64]
	}

	tags := []string{}
	for k, v := range opts.Tags {
		tags = append(tags, fmt.Sprintf("%s*%s", k, v))
	}

	params := map[string]interface{}{
		"bandwidth": map[string]interface{}{
			"name":        opts.Name,
			"size":        opts.BandwidthMbps,
			"share_type":  "PER",
			"charge_mode": opts.ChargeType,
		},
		"publicip": map[string]interface{}{
			"type":       opts.BGPType,
			"ip_version": 4,
			"alias":      opts.Name,
			"tags":       tags,
		},
	}
	if len(opts.ProjectId) > 0 {
		params["enterprise_project_id"] = opts.ProjectId
	}
	resp, err := self.post(SERVICE_VPC, "publicips", params)
	if err != nil {
		return nil, err
	}
	eip := &SEipAddress{region: self}
	return eip, resp.Unmarshal(eip, "publicip")
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EIP/doc?version=v3&api=ShowPublicip
func (self *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	resp, err := self.list(SERVICE_VPC_V3, "eip/publicips/"+eipId, nil)
	if err != nil {
		return nil, err
	}
	eip := &SEipAddress{region: self}
	return eip, resp.Unmarshal(eip, "publicip")
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EIP/doc?version=v2&api=DeletePublicip
func (self *SRegion) DeallocateEIP(eipId string) error {
	_, err := self.delete(SERVICE_VPC, "publicips/"+eipId)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EIP/doc?version=v3&api=AssociatePublicips
func (self *SRegion) AssociateEip(eipId string, associateId, associateType string) error {
	params := map[string]interface{}{
		"publicip": map[string]interface{}{
			"associate_instance_id":   associateId,
			"associate_instance_type": associateType,
		},
	}
	res := fmt.Sprintf("eip/publicips/%s/associate-instance", eipId)
	_, err := self.post(SERVICE_VPC_V3, res, params)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EIP/doc?version=v3&api=DisassociatePublicips
func (self *SRegion) DissociateEip(eipId string) error {
	res := fmt.Sprintf("eip/publicips/%s/disassociate-instance", eipId)
	_, err := self.post(SERVICE_VPC_V3, res, nil)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EIP/doc?version=v2&api=UpdateBandwidth
func (self *SRegion) UpdateEipBandwidth(bandwidthId string, bw int) error {
	params := map[string]interface{}{
		"bandwidth": map[string]interface{}{
			"size": bw,
		},
	}
	_, err := self.put(SERVICE_VPC, "bandwidths/"+bandwidthId, params)
	return err
}

func (self *SEipAddress) GetProjectId() string {
	return self.EnterpriseProjectId
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EIP/doc?version=v3&api=ListPublicips
func (self *SRegion) GetEips(portId string, addrs []string) ([]SEipAddress, error) {
	query := url.Values{}
	for _, addr := range addrs {
		query.Add("public_ip_address", addr)
	}
	if len(portId) > 0 {
		query.Set("port_id", portId)
	}
	eips := []SEipAddress{}
	for {
		resp, err := self.list(SERVICE_VPC_V3, "eip/publicips", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Publicips []SEipAddress
			PageInfo  sPageInfo
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		eips = append(eips, part.Publicips...)
		if len(part.Publicips) == 0 || len(part.PageInfo.NextMarker) == 0 {
			break
		}
		query.Set("marker", part.PageInfo.NextMarker)
	}
	for i := range eips {
		eips[i].region = self
	}
	return eips, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EIP/doc?version=v2&api=DeletePublicipTag
func (self *SRegion) DeletePublicipTag(eipId string, key string) error {
	res := fmt.Sprintf("publicips/%s/tags/%s", eipId, key)
	_, err := self.delete(SERVICE_VPC_V2_0, res)
	return err
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/EIP/doc?version=v2&api=CreatePublicipTag
func (self *SRegion) CreatePublicipTag(eipId string, tags map[string]string) error {
	params := map[string]interface{}{
		"action": "create",
	}
	add := []map[string]string{}
	for k, v := range tags {
		add = append(add, map[string]string{"key": k, "value": v})
	}
	params["tags"] = add
	res := fmt.Sprintf("publicips/%s/tags/action", eipId)
	_, err := self.post(SERVICE_VPC_V2_0, res, params)
	return err
}

func (self *SRegion) setEipTags(id string, existedTags, tags map[string]string, replace bool) error {
	deleteTagsKey := []string{}
	for k := range existedTags {
		if replace {
			deleteTagsKey = append(deleteTagsKey, k)
		} else {
			if _, ok := tags[k]; ok {
				deleteTagsKey = append(deleteTagsKey, k)
			}
		}
	}
	if len(deleteTagsKey) > 0 {
		for _, k := range deleteTagsKey {
			err := self.DeletePublicipTag(self.ID, k)
			if err != nil {
				return errors.Wrapf(err, "remove tags")
			}
		}
	}
	if len(tags) > 0 {
		err := self.CreatePublicipTag(self.ID, tags)
		if err != nil {
			return errors.Wrapf(err, "add tags")
		}
	}
	return nil
}

func (self *SEipAddress) SetTags(tags map[string]string, replace bool) error {
	existedTags, err := self.GetTags()
	if err != nil {
		return errors.Wrap(err, "self.GetTags()")
	}
	return self.region.setEipTags(self.Id, existedTags, tags, replace)
}
