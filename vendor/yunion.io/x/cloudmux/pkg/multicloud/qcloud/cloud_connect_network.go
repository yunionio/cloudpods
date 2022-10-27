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
	"fmt"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SCcn struct {
	multicloud.SResourceBase
	QcloudTags
	region *SRegion

	CcnID             string `json:"CcnId"`
	CcnName           string `json:"CcnName"`
	CcnDescription    string `json:"CcnDescription"`
	InstanceCount     int    `json:"InstanceCount"`
	CreateTime        string `json:"CreateTime"`
	State             string `json:"State"`
	AttachedInstances []SCcnInstance
}

type SCcnInstance struct {
	CcnID          string   `json:"CcnId"`
	InstanceType   string   `json:"InstanceType"`
	InstanceID     string   `json:"InstanceId"`
	InstanceName   string   `json:"InstanceName"`
	InstanceRegion string   `json:"InstanceRegion"`
	InstanceUin    string   `json:"InstanceUin"`
	CidrBlock      []string `json:"CidrBlock"`
	State          string   `json:"State"`
}

type SCcnAttachInstanceInput struct {
	InstanceType   string
	InstanceId     string
	InstanceRegion string
}

func (self *SRegion) DescribeCcns(ccnIds []string, offset int, limit int) ([]SCcn, int, error) {
	params := map[string]string{}
	params["Offset"] = strconv.Itoa(offset)
	params["Limit"] = strconv.Itoa(limit)
	if ccnIds != nil && len(ccnIds) > 0 {
		for index, ccnId := range ccnIds {
			params[fmt.Sprintf("CcnIds.%d", index)] = ccnId
		}
	}
	resp, err := self.vpcRequest("DescribeCcns", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, `self.vpcRequest("DescribeCcns", %s)`, jsonutils.Marshal(params).String())
	}
	ccns := []SCcn{}
	err = resp.Unmarshal(&ccns, "CcnSet")
	if err != nil {
		return nil, 0, errors.Wrapf(err, `(%s).Unmarshal(&ccns,"CcnSet")`, jsonutils.Marshal(resp).String())
	}
	total, _ := resp.Float("TotalCount")
	return ccns, int(total), nil
}

func (self *SRegion) GetAllCcns() ([]SCcn, error) {
	ccns := []SCcn{}
	for {
		part, total, err := self.DescribeCcns(nil, len(ccns), 50)
		if err != nil {
			return nil, errors.Wrapf(err, "self.DescribeCcns(nil, %d, 50)", len(ccns))
		}
		ccns = append(ccns, part...)
		if len(ccns) >= total {
			break
		}
	}
	for i := range ccns {
		ccns[i].region = self
	}
	return ccns, nil
}

func (self *SRegion) GetCcnById(ccnId string) (*SCcn, error) {
	ccns, _, err := self.DescribeCcns([]string{ccnId}, 0, 50)
	if err != nil {
		return nil, errors.Wrapf(err, "self.DescribeCcns(%s, 0, 50)", ccnId)
	}
	if len(ccns) < 1 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "DescribeCcns")
	}
	if len(ccns) > 1 {
		return nil, errors.Wrap(cloudprovider.ErrDuplicateId, "DescribeCcns")
	}
	ccns[0].region = self
	return &ccns[0], nil
}

func (self *SRegion) DescribeCcnAttachedInstances(ccnId string, offset int, limit int) ([]SCcnInstance, int, error) {
	params := map[string]string{}
	params["Offset"] = strconv.Itoa(offset)
	params["Limit"] = strconv.Itoa(limit)
	params["Filters.0.Name"] = "ccn-id"
	params["Filters.0.Values.0"] = ccnId
	resp, err := self.vpcRequest("DescribeCcnAttachedInstances", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, `self.vpcRequest("DescribeCcnAttachedInstances", %s)`, jsonutils.Marshal(params).String())
	}
	instances := []SCcnInstance{}
	err = resp.Unmarshal(&instances, "InstanceSet")
	if err != nil {
		return nil, 0, errors.Wrapf(err, `(%s).Unmarshal(&instances, "InstanceSet")`, jsonutils.Marshal(resp).String())
	}
	total, _ := resp.Float("TotalCount")
	return instances, int(total), nil
}

func (self *SRegion) GetAllCcnAttachedInstances(ccnId string) ([]SCcnInstance, error) {
	ccnInstances := []SCcnInstance{}
	for {
		part, total, err := self.DescribeCcnAttachedInstances(ccnId, len(ccnInstances), 50)
		if err != nil {
			return nil, errors.Wrapf(err, "self.DescribeCcns(nil, %d, 50)", len(ccnInstances))
		}
		ccnInstances = append(ccnInstances, part...)
		if len(ccnInstances) >= total {
			break
		}
	}
	return ccnInstances, nil
}

func (self *SRegion) CreateCcn(opts *cloudprovider.SInterVpcNetworkCreateOptions) (string, error) {
	params := make(map[string]string)
	params["CcnName"] = opts.Name
	params["CcnDescription"] = opts.Desc
	// 默认的后付费不能被支持
	params["InstanceChargeType"] = "PREPAID"
	// 预付费仅支持地域间限速
	params["BandwidthLimitType"] = "INTER_REGION_LIMIT"
	resp, err := self.vpcRequest("CreateCcn", params)
	if err != nil {
		return "", errors.Wrapf(err, `self.vpcRequest("CreateCcn", %s)`, jsonutils.Marshal(params).String())
	}
	ccn := &SCcn{}
	err = resp.Unmarshal(ccn, "Ccn")
	if err != nil {
		return "", errors.Wrapf(err, `(%s).Unmarshal(ccn, "Ccn")`, jsonutils.Marshal(resp).String())
	}
	return ccn.CcnID, nil
}

func (self *SRegion) AcceptAttachCcnInstances(ccnId string, instances []SCcnAttachInstanceInput) error {
	params := make(map[string]string)
	params["CcnId"] = ccnId
	for i := range instances {
		params[fmt.Sprintf("Instances.%d.InstanceType", i)] = instances[i].InstanceType
		params[fmt.Sprintf("Instances.%d.InstanceId", i)] = instances[i].InstanceId
		params[fmt.Sprintf("Instances.%d.InstanceRegion", i)] = instances[i].InstanceRegion
	}
	_, err := self.vpcRequest("AcceptAttachCcnInstances", params)
	if err != nil {
		return errors.Wrapf(err, `self.vpcRequest("AcceptAttachCcnInstances", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SRegion) DeleteCcn(ccnId string) error {
	params := make(map[string]string)
	params["CcnId"] = ccnId
	_, err := self.vpcRequest("DeleteCcn", params)
	if err != nil {
		return errors.Wrapf(err, ` self.vpcRequest("DeleteCcn", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SRegion) AttachCcnInstances(ccnId string, ccnUin string, instances []SCcnAttachInstanceInput) error {
	params := make(map[string]string)
	params["CcnId"] = ccnId
	if len(ccnUin) > 0 {
		params["CcnUin"] = ccnUin
	}
	for i := range instances {
		params[fmt.Sprintf("Instances.%d.InstanceType", i)] = instances[i].InstanceType
		params[fmt.Sprintf("Instances.%d.InstanceId", i)] = instances[i].InstanceId
		params[fmt.Sprintf("Instances.%d.InstanceRegion", i)] = instances[i].InstanceRegion
	}

	_, err := self.vpcRequest("AttachCcnInstances", params)
	if err != nil {
		return errors.Wrapf(err, ` self.vpcRequest("AttachCcnInstances", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SRegion) DetachCcnInstances(ccnId string, instances []SCcnAttachInstanceInput) error {
	params := make(map[string]string)
	params["CcnId"] = ccnId

	for i := range instances {
		params[fmt.Sprintf("Instances.%d.InstanceType", i)] = instances[i].InstanceType
		params[fmt.Sprintf("Instances.%d.InstanceId", i)] = instances[i].InstanceId
		params[fmt.Sprintf("Instances.%d.InstanceRegion", i)] = instances[i].InstanceRegion
	}

	_, err := self.vpcRequest("DetachCcnInstances", params)
	if err != nil {
		return errors.Wrapf(err, ` self.vpcRequest("DetachCcnInstances", %s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SCcn) GetId() string {
	return self.CcnID
}

func (self *SCcn) GetName() string {
	return self.CcnName
}

func (self *SCcn) GetGlobalId() string {
	return self.GetId()
}

func (self *SCcn) GetStatus() string {
	switch self.State {
	case "AVAILABLE":
		return api.INTER_VPC_NETWORK_STATUS_AVAILABLE
	case "ISOLATED":
		return api.INTER_VPC_NETWORK_STATUS_UNKNOWN
	default:
		return api.INTER_VPC_NETWORK_STATUS_UNKNOWN
	}
}

func (self *SCcn) Refresh() error {
	ccn, err := self.region.GetCcnById(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "self.region.GetCcnById(%s)", self.GetId())
	}
	return jsonutils.Update(self, ccn)
}

func (self *SCcn) GetAuthorityOwnerId() string {
	return self.region.client.ownerName
}

func (self *SCcn) fetchAttachedInstances() error {
	if self.AttachedInstances != nil {
		return nil
	}

	instances, err := self.region.GetAllCcnAttachedInstances(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "self.region.GetAllCcnAttachedInstances(%s)", self.GetId())
	}
	self.AttachedInstances = instances
	return nil
}

func (self *SCcn) GetICloudVpcIds() ([]string, error) {
	err := self.fetchAttachedInstances()
	if err != nil {
		return nil, errors.Wrap(err, "self.fetchAttachedInstances()")
	}
	result := []string{}
	for i := range self.AttachedInstances {
		if self.AttachedInstances[i].InstanceType == "VPC" {
			result = append(result, self.AttachedInstances[i].InstanceID)
		}
	}
	return result, nil
}

func (self *SCcn) AttachVpc(opts *cloudprovider.SInterVpcNetworkAttachVpcOption) error {
	if self.GetAuthorityOwnerId() == opts.VpcAuthorityOwnerId {
		return nil
	}
	instance := SCcnAttachInstanceInput{
		InstanceType:   "VPC",
		InstanceId:     opts.VpcId,
		InstanceRegion: opts.VpcRegionId,
	}
	err := self.region.AcceptAttachCcnInstances(self.GetId(), []SCcnAttachInstanceInput{instance})
	if err != nil {
		return errors.Wrapf(err, "self.region.AcceptAttachCcnInstance(%s,%s)", self.GetId(), jsonutils.Marshal(opts).String())
	}
	return nil
}

func (self *SCcn) DetachVpc(opts *cloudprovider.SInterVpcNetworkDetachVpcOption) error {
	instance := SCcnAttachInstanceInput{
		InstanceType:   "VPC",
		InstanceId:     opts.VpcId,
		InstanceRegion: opts.VpcRegionId,
	}
	err := self.region.DetachCcnInstances(self.GetId(), []SCcnAttachInstanceInput{instance})
	if err != nil {
		return errors.Wrapf(err, "self.client.DetachCcnInstance(%s,%s)", self.GetId(), jsonutils.Marshal(opts).String())
	}
	return nil
}

func (self *SCcn) Delete() error {
	err := self.region.DeleteCcn(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "self.region.DeleteCcn(%s)", self.GetId())
	}
	return nil
}

func (self *SCcn) GetIRoutes() ([]cloudprovider.ICloudInterVpcNetworkRoute, error) {
	routes, err := self.region.GetAllCcnRouteSets(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, " self.region.GetAllCcnRouteSets(self.GetId())")
	}
	result := []cloudprovider.ICloudInterVpcNetworkRoute{}
	for i := range routes {
		result = append(result, &routes[i])
	}
	return result, nil
}

func (self *SCcn) EnableRouteEntry(routeId string) error {
	err := self.region.EnableCcnRoutes(self.GetId(), []string{routeId})
	if err != nil {
		return errors.Wrap(err, "self.region.EnableCcnRoutes")
	}
	return nil
}

func (self *SCcn) DisableRouteEntry(routeId string) error {
	err := self.region.DisableCcnRoutes(self.GetId(), []string{routeId})
	if err != nil {
		return errors.Wrap(err, "self.region.DisableCcnRoutes")
	}
	return nil
}

func (client *SQcloudClient) GetICloudInterVpcNetworks() ([]cloudprovider.ICloudInterVpcNetwork, error) {
	region, err := client.getDefaultRegion()
	if err != nil {
		return nil, errors.Wrap(err, "getDefaultRegion")
	}
	sccns, err := region.GetAllCcns()
	if err != nil {
		return nil, errors.Wrap(err, " region.GetAllCcns()")
	}
	result := []cloudprovider.ICloudInterVpcNetwork{}
	for i := range sccns {
		result = append(result, &sccns[i])
	}
	return result, nil
}

func (client *SQcloudClient) GetICloudInterVpcNetworkById(id string) (cloudprovider.ICloudInterVpcNetwork, error) {
	region, err := client.getDefaultRegion()
	if err != nil {
		return nil, errors.Wrap(err, "getDefaultRegion")
	}
	ccn, err := region.GetCcnById(id)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetCcnById(%s)", id)
	}
	return ccn, nil
}

func (client *SQcloudClient) CreateICloudInterVpcNetwork(opts *cloudprovider.SInterVpcNetworkCreateOptions) (cloudprovider.ICloudInterVpcNetwork, error) {
	region, err := client.getDefaultRegion()
	if err != nil {
		return nil, errors.Wrap(err, "getDefaultRegion")
	}

	ccnId, err := region.CreateCcn(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "region.CreateCcn(%s)", jsonutils.Marshal(opts).String())
	}
	iNetwork, err := client.GetICloudInterVpcNetworkById(ccnId)
	if err != nil {
		return nil, errors.Wrapf(err, "client.GetICloudInterVpcNetworkById(%s)", ccnId)
	}
	return iNetwork, nil
}
