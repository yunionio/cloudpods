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
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SCen struct {
	multicloud.SResourceBase
	AliyunTags
	client                 *SAliyunClient
	Status                 string                 `json:"Status"`
	ProtectionLevel        string                 `json:"ProtectionLevel"`
	CenID                  string                 `json:"CenId"`
	CreationTime           string                 `json:"CreationTime"`
	CenBandwidthPackageIds CenBandwidthPackageIds `json:"CenBandwidthPackageIds"`
	Name                   string                 `json:"Name"`
}

type SCens struct {
	TotalCount int    `json:"TotalCount"`
	RequestID  string `json:"RequestId"`
	PageSize   int    `json:"PageSize"`
	PageNumber int    `json:"PageNumber"`
	Cens       sCens  `json:"Cens"`
}

type CenBandwidthPackageIds struct {
	CenBandwidthPackageID []string `json:"CenBandwidthPackageId"`
}

type sCens struct {
	Cen []SCen `json:"Cen"`
}

type SCenChildInstances struct {
	PageNumber     int                `json:"PageNumber"`
	ChildInstances sCenChildInstances `json:"ChildInstances"`
	TotalCount     int                `json:"TotalCount"`
	PageSize       int                `json:"PageSize"`
	RequestID      string             `json:"RequestId"`
}

type SCenChildInstance struct {
	Status                string `json:"Status"`
	ChildInstanceOwnerID  string `json:"ChildInstanceOwnerId"`
	ChildInstanceID       string `json:"ChildInstanceId"`
	ChildInstanceRegionID string `json:"ChildInstanceRegionId"`
	CenID                 string `json:"CenId"`
	ChildInstanceType     string `json:"ChildInstanceType"`
}

type sCenChildInstances struct {
	ChildInstance []SCenChildInstance `json:"ChildInstance"`
}

type SCenAttachInstanceInput struct {
	InstanceType         string
	InstanceId           string
	InstanceRegion       string
	ChildInstanceOwnerId string
}

func (client *SAliyunClient) DescribeCens(pageNumber int, pageSize int) (SCens, error) {
	scens := SCens{}
	params := map[string]string{}
	params["Action"] = "DescribeCens"
	params["PageNumber"] = strconv.Itoa(pageNumber)
	params["PageSize"] = strconv.Itoa(pageSize)
	resp, err := client.cbnRequest("DescribeCens", params)
	if err != nil {
		return scens, errors.Wrap(err, "DescribeCens")
	}
	err = resp.Unmarshal(&scens)
	if err != nil {
		return scens, errors.Wrap(err, "resp.Unmarshal")
	}
	return scens, nil
}

func (client *SAliyunClient) GetAllCens() ([]SCen, error) {
	pageNumber := 0
	sCen := []SCen{}
	for {
		pageNumber++
		cens, err := client.DescribeCens(pageNumber, 20)
		if err != nil {
			return nil, errors.Wrapf(err, "client.DescribeCens(%d, 20)", pageNumber)
		}
		sCen = append(sCen, cens.Cens.Cen...)
		if len(sCen) >= cens.TotalCount {
			break
		}
	}
	for i := 0; i < len(sCen); i++ {
		sCen[i].client = client
	}
	return sCen, nil
}

func (client *SAliyunClient) CreateCen(opts *cloudprovider.SInterVpcNetworkCreateOptions) (string, error) {
	params := map[string]string{}
	params["Name"] = opts.Name
	params["Description"] = opts.Desc
	resp, err := client.cbnRequest("CreateCen", params)
	if err != nil {
		return "", errors.Wrap(err, "CreateCen")
	}
	type CentId struct {
		CenId string `json:"CenId"`
	}
	centId := CentId{}
	err = resp.Unmarshal(&centId)
	if err != nil {
		return "", errors.Wrap(err, "resp.Unmarshal")
	}
	return centId.CenId, nil
}

func (client *SAliyunClient) DeleteCen(id string) error {
	params := map[string]string{}
	params["CenId"] = id
	_, err := client.cbnRequest("DeleteCen", params)
	if err != nil {
		return errors.Wrap(err, "DeleteCen")
	}
	return nil
}

func (client *SAliyunClient) DescribeCenAttachedChildInstances(cenId string, pageNumber int, pageSize int) (SCenChildInstances, error) {
	scenChilds := SCenChildInstances{}
	params := map[string]string{}
	params["CenId"] = cenId
	params["PageNumber"] = strconv.Itoa(pageNumber)
	params["PageSize"] = strconv.Itoa(pageSize)
	resp, err := client.cbnRequest("DescribeCenAttachedChildInstances", params)
	if err != nil {
		return scenChilds, errors.Wrap(err, "DescribeCenAttachedChildInstances")
	}
	err = resp.Unmarshal(&scenChilds)
	if err != nil {
		return scenChilds, errors.Wrap(err, "resp.Unmarshal")
	}
	return scenChilds, nil
}

func (client *SAliyunClient) GetAllCenAttachedChildInstances(cenId string) ([]SCenChildInstance, error) {
	pageNumber := 0
	scenChilds := []SCenChildInstance{}
	for {
		pageNumber++
		cenChilds, err := client.DescribeCenAttachedChildInstances(cenId, pageNumber, 20)
		if err != nil {
			return nil, errors.Wrapf(err, "client.DescribeCens(%d, 20)", pageNumber)
		}
		scenChilds = append(scenChilds, cenChilds.ChildInstances.ChildInstance...)
		if len(scenChilds) >= cenChilds.TotalCount {
			break
		}
	}
	return scenChilds, nil
}

func (client *SAliyunClient) AttachCenChildInstance(cenId string, instance SCenAttachInstanceInput) error {
	params := map[string]string{}
	params["CenId"] = cenId
	params["ChildInstanceId"] = instance.InstanceId
	params["ChildInstanceRegionId"] = instance.InstanceRegion
	params["ChildInstanceType"] = instance.InstanceType
	params["ChildInstanceOwnerId"] = instance.ChildInstanceOwnerId
	_, err := client.cbnRequest("AttachCenChildInstance", params)
	if err != nil {
		return errors.Wrap(err, "AttachCenChildInstance")
	}
	return nil
}

func (client *SAliyunClient) DetachCenChildInstance(cenId string, instance SCenAttachInstanceInput) error {
	params := map[string]string{}
	params["CenId"] = cenId
	params["ChildInstanceId"] = instance.InstanceId
	params["ChildInstanceRegionId"] = instance.InstanceRegion
	params["ChildInstanceType"] = instance.InstanceType
	params["ChildInstanceOwnerId"] = instance.ChildInstanceOwnerId
	_, err := client.cbnRequest("DetachCenChildInstance", params)
	if err != nil {
		return errors.Wrap(err, "DetachCenChildInstance")
	}
	return nil
}

func (self *SCen) GetId() string {
	return self.CenID
}

func (self *SCen) GetName() string {
	return self.Name
}

func (self *SCen) GetGlobalId() string {
	return self.GetId()
}

func (self *SCen) GetStatus() string {
	switch self.Status {
	case "Creating":
		return api.INTER_VPC_NETWORK_STATUS_CREATING
	case "Active":
		return api.INTER_VPC_NETWORK_STATUS_AVAILABLE
	case "Deleting":
		return api.INTER_VPC_NETWORK_STATUS_DELETING
	default:
		return api.INTER_VPC_NETWORK_STATUS_UNKNOWN
	}
}

func (self *SCen) Refresh() error {
	scens, err := self.client.GetAllCens()
	if err != nil {
		return errors.Wrap(err, "self.client.GetAllCens()")
	}
	for i := range scens {
		if scens[i].CenID == self.CenID {
			return jsonutils.Update(self, scens[i])
		}
	}
	return cloudprovider.ErrNotFound
}

func (self *SCen) GetAuthorityOwnerId() string {
	return self.client.ownerId
}

func (self *SCen) GetICloudVpcIds() ([]string, error) {
	childs, err := self.client.GetAllCenAttachedChildInstances(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "self.client.GetAllCenAttachedChildInstances(self.GetId())")
	}
	vpcIds := []string{}
	for i := range childs {
		if childs[i].ChildInstanceType == "VPC" {
			vpcIds = append(vpcIds, childs[i].ChildInstanceID)
		}
	}
	return vpcIds, nil
}

func (self *SCen) AttachVpc(opts *cloudprovider.SInterVpcNetworkAttachVpcOption) error {
	instance := SCenAttachInstanceInput{
		InstanceType:         "VPC",
		InstanceId:           opts.VpcId,
		InstanceRegion:       opts.VpcRegionId,
		ChildInstanceOwnerId: opts.VpcAuthorityOwnerId,
	}
	err := self.client.AttachCenChildInstance(self.GetId(), instance)
	if err != nil {
		return errors.Wrapf(err, "self.client.AttachCenChildInstance(%s,%s)", self.GetId(), jsonutils.Marshal(opts).String())
	}
	return nil
}

func (self *SCen) DetachVpc(opts *cloudprovider.SInterVpcNetworkDetachVpcOption) error {
	instance := SCenAttachInstanceInput{
		InstanceType:         "VPC",
		InstanceId:           opts.VpcId,
		InstanceRegion:       opts.VpcRegionId,
		ChildInstanceOwnerId: opts.VpcAuthorityOwnerId,
	}
	err := self.client.DetachCenChildInstance(self.GetId(), instance)
	if err != nil {
		return errors.Wrapf(err, "self.client.DetachCenChildInstance(%s,%s)", self.GetId(), jsonutils.Marshal(opts).String())
	}
	return nil
}

func (self *SCen) Delete() error {
	err := self.client.DeleteCen(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "self.client.DeleteCen(%s)", self.GetId())
	}
	return nil
}

func (self *SCen) GetInstanceRouteEntries() ([]SCenRouteEntry, error) {
	childInstance, err := self.client.GetAllCenAttachedChildInstances(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "self.client.GetAllCenAttachedChildInstances(self.GetId())")
	}
	result := []SCenRouteEntry{}
	for i := range childInstance {
		routes, err := self.client.GetAllCenChildInstanceRouteEntries(self.GetId(), childInstance[i].ChildInstanceID, childInstance[i].ChildInstanceRegionID, childInstance[i].ChildInstanceType)
		if err != nil {
			return nil, errors.Wrap(err, "self.client.GetAllCenChildInstanceRouteEntries(self.GetId(), childInstance[i].ChildInstanceID, childInstance[i].ChildInstanceRegionID, childInstance[i].ChildInstanceType)")
		}
		for j := range routes {
			// CEN 类型的路由是通过CEN从其他vpc/vbr的路由表中传播过来的
			// 只关注路由的发源
			if routes[j].Type != "CEN" {
				routes[j].ChildInstance = &childInstance[i]
				result = append(result, routes[j])
			}
		}
	}
	return result, nil
}

func (self *SCen) GetIRoutes() ([]cloudprovider.ICloudInterVpcNetworkRoute, error) {
	result := []cloudprovider.ICloudInterVpcNetworkRoute{}
	routeEntries, err := self.GetInstanceRouteEntries()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetInstanceRouteEntries()")
	}
	for i := range routeEntries {
		result = append(result, &routeEntries[i])
	}
	return result, nil
}

func (self *SCen) EnableRouteEntry(routeId string) error {
	idContent := strings.Split(routeId, ":")
	if len(idContent) != 2 {
		return errors.Wrapf(cloudprovider.ErrNotSupported, "invalid aliyun generated cenRouteId %s", routeId)
	}
	routeTable := idContent[0]
	cidr := idContent[1]
	routeEntries, err := self.GetInstanceRouteEntries()
	if err != nil {
		return errors.Wrap(err, "self.GetInstanceRouteEntries()")
	}
	routeEntry := SCenRouteEntry{}
	for i := range routeEntries {
		if routeEntries[i].RouteTableID == routeTable && routeEntries[i].DestinationCidrBlock == cidr {
			routeEntry = routeEntries[i]
			break
		}
	}
	if routeEntry.GetEnabled() {
		return nil
	}
	err = self.client.PublishRouteEntries(self.GetId(), routeEntry.GetInstanceId(), routeTable, routeEntry.GetInstanceRegionId(), routeEntry.GetInstanceType(), cidr)
	if err != nil {
		return errors.Wrap(err, "self.client.PublishRouteEntries()")
	}
	return nil
}

func (self *SCen) DisableRouteEntry(routeId string) error {
	idContent := strings.Split(routeId, ":")
	if len(idContent) != 2 {
		return errors.Wrapf(cloudprovider.ErrNotSupported, "invalid aliyun generated cenRouteId %s", routeId)
	}
	routeTable := idContent[0]
	cidr := idContent[1]
	routeEntries, err := self.GetInstanceRouteEntries()
	if err != nil {
		return errors.Wrap(err, "self.GetInstanceRouteEntries()")
	}
	routeEntry := SCenRouteEntry{}
	for i := range routeEntries {
		if routeEntries[i].RouteTableID == routeTable && routeEntries[i].DestinationCidrBlock == cidr {
			routeEntry = routeEntries[i]
			break
		}
	}
	if !routeEntry.GetEnabled() {
		return nil
	}
	err = self.client.WithdrawPublishedRouteEntries(self.GetId(), routeEntry.GetInstanceId(), routeTable, routeEntry.GetInstanceRegionId(), routeEntry.GetInstanceType(), cidr)
	if err != nil {
		return errors.Wrap(err, "self.client.PublishRouteEntries()")
	}
	return nil
}
