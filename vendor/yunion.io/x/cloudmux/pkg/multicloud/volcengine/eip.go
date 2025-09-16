// Copyright 2023 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package volcengine

import (
	"fmt"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type TBillingType int

const (
	BillingByPrePaid TBillingType = iota + 1
	BillingByBandwidth
	BillingByTraffic
)

const (
	EIP_STATUS_ATTACHING = "Attaching"
	EIP_STATUS_DETACHING = "Detaching"
	EIP_STATUS_ATTACHED  = "Attached"
	EIP_STATUS_AVAILABLE = "Available"
	EIP_STATUS_DELETING  = "Deleting"

	EIP_INSTANCE_TYPE_NAT   = "Nat"              // NAT网关
	EIP_INSTANCE_TYPE_ENI   = "NetworkInterface" // 辅助网卡
	EIP_INSTANCE_TYPE_CLB   = "ClbInstance"      // 负载均衡
	EIP_INSTANCE_TYPE_ALB   = "Albinstance"      // 应用型负载均衡
	EIP_INSTANCE_TYPE_ECS   = "EcsInstance"      // 云服务器
	EIP_INSTANCE_TYPE_HAVIP = "HaVip"            // 高可用虚拟IP
)

type SEipAddress struct {
	region *SRegion
	multicloud.SEipBase
	VolcEngineTags

	Name         string
	AllocationId string

	BillingType TBillingType

	EipAddress string
	Status     string

	InstanceType string
	InstanceId   string
	Bandwidth    int /* Mbps */

	BusinessStatus string
	AllocationTime time.Time
	Description    string
	ISP            string

	LockReason string

	ExpiredTime time.Time
	ProjectName string
}

func (eipaddr *SEipAddress) GetId() string {
	return eipaddr.AllocationId
}

func (eipaddr *SEipAddress) GetName() string {
	if len(eipaddr.Name) > 0 {
		return eipaddr.Name
	}
	return eipaddr.EipAddress
}

func (eipaddr *SEipAddress) GetGlobalId() string {
	return eipaddr.AllocationId
}

func (eipaddr *SEipAddress) GetStatus() string {
	switch eipaddr.Status {
	case EIP_STATUS_ATTACHED, EIP_STATUS_AVAILABLE:
		return api.EIP_STATUS_READY
	case EIP_STATUS_ATTACHING:
		return api.EIP_STATUS_ASSOCIATE
	case EIP_STATUS_DETACHING:
		return api.EIP_STATUS_DISSOCIATE
	case EIP_STATUS_DELETING:
		return api.EIP_STATUS_DEALLOCATE
	default:
		return api.EIP_STATUS_UNKNOWN
	}
}

func (eipaddr *SEipAddress) Refresh() error {
	if eipaddr.IsEmulated() {
		return nil
	}
	new, err := eipaddr.region.GetEip(eipaddr.AllocationId)
	if err != nil {
		return err
	}
	return jsonutils.Update(eipaddr, new)
}

func (eipaddr *SEipAddress) GetProjectId() string {
	return eipaddr.ProjectName
}

func (eipaddr *SEipAddress) GetIpAddr() string {
	return eipaddr.EipAddress
}

func (eipaddr *SEipAddress) GetMode() string {
	if eipaddr.InstanceId == eipaddr.AllocationId {
		return api.EIP_MODE_INSTANCE_PUBLICIP
	} else {
		return api.EIP_MODE_STANDALONE_EIP
	}
}

func (eipaddr *SEipAddress) GetINetworkId() string {
	return ""
}

func (eipaddr *SEipAddress) GetAssociationType() string {
	switch eipaddr.InstanceType {
	case EIP_INSTANCE_TYPE_ECS, EIP_INSTANCE_TYPE_ENI:
		return api.EIP_ASSOCIATE_TYPE_SERVER
	case EIP_INSTANCE_TYPE_NAT:
		return api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY
	case EIP_INSTANCE_TYPE_ALB, EIP_INSTANCE_TYPE_CLB:
		return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
	default:
		return eipaddr.InstanceType
	}
}

func (eipaddr *SEipAddress) GetAssociationExternalId() string {
	return eipaddr.InstanceId
}

func (eipaddr *SEipAddress) GetBandwidth() int {
	return eipaddr.Bandwidth
}

func (eipaddr *SEipAddress) GetInternetChargeType() string {
	switch eipaddr.BillingType {
	case BillingByPrePaid, BillingByBandwidth:
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	case BillingByTraffic:
		return api.EIP_CHARGE_TYPE_BY_TRAFFIC
	default:
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	}
}

func (eipaddr *SEipAddress) Delete() error {
	return eipaddr.region.DeallocateEIP(eipaddr.AllocationId)
}

func (eipaddr *SEipAddress) Associate(conf *cloudprovider.AssociateConfig) error {
	_ = cloudprovider.Wait(20*time.Second, time.Minute, func() (bool, error) {
		err := eipaddr.region.AssociateEip(eipaddr.AllocationId, conf.InstanceId)
		if err != nil {
			if isError(err, "IncorrectInstanceStatus") {
				return false, nil
			}
			return false, errors.Wrapf(err, "region.AssociateEip")
		}
		return true, nil
	})
	err := cloudprovider.WaitStatus(eipaddr, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (eipaddr *SEipAddress) Dissociate() error {
	err := eipaddr.region.DissociateEip(eipaddr.AllocationId, eipaddr.InstanceId)
	if err != nil {
		return err
	}
	err = cloudprovider.WaitStatus(eipaddr, api.EIP_STATUS_READY, 10*time.Second, 180*time.Second)
	return err
}

func (eipaddr *SEipAddress) ChangeBandwidth(bw int) error {
	return eipaddr.region.UpdateEipBandwidth(eipaddr.AllocationId, bw)
}

func getInstanceType(instanceId string) (string, error) {
	prefixMap := map[string]string{
		"i-":     EIP_INSTANCE_TYPE_ECS,
		"clb-":   EIP_INSTANCE_TYPE_CLB,
		"alb-":   EIP_INSTANCE_TYPE_ALB,
		"ngw-":   EIP_INSTANCE_TYPE_NAT,
		"eni-":   EIP_INSTANCE_TYPE_ENI,
		"havip-": EIP_INSTANCE_TYPE_HAVIP,
	}
	for prefix, instanceType := range prefixMap {
		if strings.HasPrefix(instanceId, prefix) {
			return instanceType, nil
		}
	}
	return "", errors.Errorf("Unknown instance type for %s", instanceId)
}

func (region *SRegion) GetEips(eipIds []string, associatedId string, addresses []string, pageNumber int, pageSize int) ([]SEipAddress, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}

	params := make(map[string]string)
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)

	for index, addr := range addresses {
		params[fmt.Sprintf("EipAddresses.%d", index+1)] = addr
	}

	for index, eipId := range eipIds {
		params[fmt.Sprintf("AllocationIds.%d", index+1)] = eipId
	}

	if len(associatedId) > 0 {
		params["AssociatedInstanceId"] = associatedId
		associatedType, err := getInstanceType(associatedId)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "Unknown associated type")
		}
		params["AssociatedInstanceType"] = associatedType
	}

	body, err := region.vpcRequest("DescribeEipAddresses", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeEipAddresses fail")
	}

	eips := make([]SEipAddress, 0)
	err = body.Unmarshal(&eips, "EipAddresses")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal EipAddress details fail")
	}
	total, _ := body.Int("TotalCount")
	for i := 0; i < len(eips); i += 1 {
		eips[i].region = region
	}
	return eips, int(total), nil
}

func (region *SRegion) GetEip(eipId string) (*SEipAddress, error) {
	eips, _, err := region.GetEips([]string{eipId}, "", make([]string, 0), 1, 1)
	if err != nil {
		return nil, err
	}
	for i := range eips {
		if eips[i].AllocationId == eipId {
			eips[i].region = region
			return &eips[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, eipId)
}

func (region *SRegion) AllocateEIP(opts *cloudprovider.SEip) (*SEipAddress, error) {
	params := make(map[string]string)
	if len(opts.Name) > 0 {
		params["Name"] = opts.Name
	}
	params["Bandwidth"] = fmt.Sprintf("%d", opts.BandwidthMbps)
	switch opts.ChargeType {
	case api.EIP_CHARGE_TYPE_BY_TRAFFIC:
		params["BillingType"] = fmt.Sprintf("%d", BillingByTraffic)
	case api.EIP_CHARGE_TYPE_BY_BANDWIDTH:
		params["BillingType"] = fmt.Sprintf("%d", BillingByBandwidth)
	}

	params["ClientToken"] = utils.GenRequestId(20)
	if len(opts.ProjectId) > 0 {
		params["ProjectName"] = opts.ProjectId
	}
	params["ISP"] = "BGP"

	index := 1
	for key, value := range opts.Tags {
		params[fmt.Sprintf("Tags.%d.Key", index)] = key
		params[fmt.Sprintf("Tags.%d.Value", index)] = value
		index++
	}

	body, err := region.vpcRequest("AllocateEipAddress", params)
	if err != nil {
		return nil, errors.Wrapf(err, "AllocateEipAddress fail")
	}

	eipId, err := body.GetString("AllocationId")
	if err != nil {
		return nil, errors.Wrapf(err, "get AllocationId after created fail")
	}

	err = cloudprovider.Wait(5*time.Second, time.Minute, func() (bool, error) {
		_, err := region.GetEip(eipId)
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		} else {
			return true, err
		}
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find eip after create")
	}
	return region.GetEip(eipId)
}

func (region *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return region.AllocateEIP(opts)
}

func (region *SRegion) DeallocateEIP(eipId string) error {
	params := make(map[string]string)
	params["AllocationId"] = eipId

	_, err := region.vpcRequest("ReleaseEipAddress", params)
	if err != nil {
		err = errors.Wrapf(err, "ReleaseEipAddress fail")
	}
	return err
}

func (region *SRegion) AssociateEip(eipId string, instanceId string) error {
	params := make(map[string]string)
	params["AllocationId"] = eipId
	params["InstanceId"] = instanceId

	instanceType, err := getInstanceType(instanceId)
	if err != nil {
		return errors.Wrapf(err, "Unknown instance type")
	}
	params["InstanceType"] = instanceType

	_, err = region.vpcRequest("AssociateEipAddress", params)
	return errors.Wrapf(err, "AssociateEipAddress fail")
}

func (region *SRegion) DissociateEip(eipId string, instanceId string) error {
	params := make(map[string]string)
	params["AllocationId"] = eipId
	params["InstanceId"] = instanceId

	instanceType, err := getInstanceType(instanceId)
	if err != nil {
		return errors.Wrapf(err, "Unknown instance type")
	}
	params["InstanceType"] = instanceType

	_, err = region.vpcRequest("DisassociateEipAddress", params)
	if err != nil {
		err = errors.Wrapf(err, "DisassociateEipAddress fail")
	}
	return err
}

func (region *SRegion) UpdateEipBandwidth(eipId string, bw int) error {
	params := make(map[string]string)
	params["AllocationId"] = eipId
	params["Bandwidth"] = fmt.Sprintf("%d", bw)

	_, err := region.vpcRequest("ModifyEipAddressAttributes", params)
	if err != nil {
		err = errors.Wrapf(err, "ModifyEipAddressAttributes fail")
	}
	return err
}
