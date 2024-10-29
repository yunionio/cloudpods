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

package ctyun

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SEip struct {
	region *SRegion
	multicloud.SEipBase
	CtyunTags

	Id               string
	Name             string
	EipAddress       string
	AssociationId    string
	AssociationType  string
	PrivateIPAddress string
	Bandwidth        int
	Status           string
	Tags             string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ExpiredAt        time.Time
	ProjectId        string
}

func (self *SEip) GetBillingType() string {
	if !self.ExpiredAt.IsZero() {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SEip) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SEip) GetExpiredAt() time.Time {
	return self.ExpiredAt
}

func (self *SEip) GetId() string {
	return self.Id
}

func (self *SEip) GetName() string {
	return self.Name
}

func (self *SEip) GetGlobalId() string {
	return self.GetId()
}

func (self *SEip) GetStatus() string {
	switch self.Status {
	case "ACTIVE", "DOWN", "UPDATING", "BANDING_OR_UNBANGDING":
		return api.EIP_STATUS_READY
	case "ERROR":
		return api.EIP_STATUS_ALLOCATE_FAIL
	case "DELETING", "DELETED", "EXPIRED":
		return api.EIP_STATUS_DEALLOCATE
	default:
		return strings.ToLower(self.Status)
	}
}

func (self *SEip) Refresh() error {
	eip, err := self.region.GetEip(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, eip)
}

func (self *SEip) GetProjectId() string {
	return ""
}

func (self *SEip) GetIpAddr() string {
	return self.EipAddress
}

func (self *SEip) GetMode() string {
	return api.EIP_MODE_STANDALONE_EIP
}

func (self *SEip) GetAssociationType() string {
	switch self.AssociationType {
	case "LOADBALANCER":
		return api.EIP_ASSOCIATE_TYPE_LOADBALANCER
	case "INSTANCE":
		return api.EIP_ASSOCIATE_TYPE_SERVER
	default:
		return strings.ToLower(self.AssociationType)
	}
}

func (self *SEip) GetAssociationExternalId() string {
	return self.AssociationId
}

func (self *SEip) GetBandwidth() int {
	return self.Bandwidth
}

func (self *SEip) GetInternetChargeType() string {
	if self.GetBillingType() == billing_api.BILLING_TYPE_PREPAID {
		return api.EIP_CHARGE_TYPE_BY_BANDWIDTH
	}
	return api.EIP_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SEip) Delete() error {
	return self.region.DeleteEip(self.GetId())
}

func (self *SEip) Associate(opts *cloudprovider.AssociateConfig) error {
	return self.region.AssociateEip(self.GetId(), opts.InstanceId)
}

func (self *SEip) Dissociate() error {
	return self.region.DissociateEip(self.GetId())
}

func (self *SEip) ChangeBandwidth(bw int) error {
	return self.region.ChangeBandwidthEip(self.GetId(), bw)
}

func (self *SRegion) GetEips(status string) ([]SEip, error) {
	pageNo := 1
	params := map[string]interface{}{
		"pageNo":   pageNo,
		"pageSize": 50,
	}
	if len(status) > 0 {
		params["status"] = status
	}
	ret := []SEip{}
	for {
		params["clientToken"] = utils.GenRequestId(20)
		resp, err := self.post(SERVICE_VPC, "/v4/eip/list", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			ReturnObj struct {
				Eips []SEip
			}
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ReturnObj.Eips...)
		if len(ret) >= part.TotalCount || len(part.ReturnObj.Eips) == 0 {
			break
		}
		pageNo++
		params["pageNo"] = pageNo
	}
	return ret, nil
}

func (self *SRegion) GetEip(eipId string) (*SEip, error) {
	params := map[string]interface{}{
		"eipID": eipId,
	}
	resp, err := self.list(SERVICE_VPC, "/v4/eip/show", params)
	if err != nil {
		return nil, err
	}
	ret := &SEip{region: self}
	err = resp.Unmarshal(ret, "returnObj")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SRegion) CreateEip(opts *cloudprovider.SEip) (*SEip, error) {
	params := map[string]interface{}{
		"clientToken":       utils.GenRequestId(20),
		"name":              opts.Name,
		"cycleType":         "on_demand",
		"demandBillingType": "upflowc",
	}
	if opts.BandwidthMbps > 0 {
		params["bandwidth"] = opts.BandwidthMbps
	}
	if opts.ChargeType == "bandwidth" {
		params["demandBillingType"] = "bandwidth"
	}
	var err error
	var eipId string
	for i := 0; i < 10; i++ {
		resp, err := self.post(SERVICE_VPC, "/v4/eip/create", params)
		if err != nil {
			return nil, err
		}
		status, err := resp.GetString("returnObj", "masterResourceStatus")
		if err != nil {
			return nil, errors.Wrapf(err, "get resource status")
		}
		if status != "started" {
			time.Sleep(time.Second * 5)
			continue
		}
		eipId, err = resp.GetString("returnObj", "eipID")
		if len(eipId) > 0 {
			break
		}
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get eipId")
	}
	return self.GetEip(eipId)
}

func (self *SRegion) DeleteEip(id string) error {
	params := map[string]interface{}{
		"clientToken": utils.GenRequestId(20),
		"eipID":       id,
	}
	_, err := self.post(SERVICE_VPC, "/v4/eip/delete", params)
	return err
}

func (self *SRegion) AssociateEip(eipId, instanceId string) error {
	params := map[string]interface{}{
		"clientToken":     utils.GenRequestId(20),
		"eipID":           eipId,
		"associationID":   instanceId,
		"associationType": 1,
	}
	_, err := self.post(SERVICE_VPC, "/v4/eip/associate", params)
	return err
}

func (self *SRegion) DissociateEip(eipId string) error {
	params := map[string]interface{}{
		"clientToken": utils.GenRequestId(20),
		"eipID":       eipId,
	}
	_, err := self.post(SERVICE_VPC, "/v4/eip/disassociate", params)
	return err
}

func (self *SRegion) ChangeBandwidthEip(eipId string, bandwidth int) error {
	params := map[string]interface{}{
		"clientToken": utils.GenRequestId(20),
		"eipID":       eipId,
		"bandwidth":   bandwidth,
	}
	_, err := self.post(SERVICE_VPC, "/v4/eip/modify-spec", params)
	return err
}
