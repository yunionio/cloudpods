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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var allowedTtls = []int64{5, 10, 15, 20, 30, 60, 120, 300, 600, 1800, 3600, 43200, 86400}

type SPvtzVpc struct {
	VpcID      string `json:"VpcId"`
	RegionName string `json:"RegionName"`
	VpcName    string `json:"VpcName"`
	RegionID   string `json:"RegionId"`
}

type SPvtzBindVpcs struct {
	Vpc []SPvtzVpc `json:"Vpc"`
}

type SPrivateZone struct {
	multicloud.SResourceBase
	AliyunTags
	client *SAliyunClient

	// RequestID       string        `json:"RequestId"`
	ZoneID          string        `json:"ZoneId"`
	SlaveDNS        string        `json:"SlaveDns"`
	ResourceGroupID string        `json:"ResourceGroupId"`
	ProxyPattern    string        `json:"ProxyPattern"`
	CreateTime      string        `json:"CreateTime"`
	Remark          string        `json:"Remark"`
	ZoneName        string        `json:"ZoneName"`
	UpdateTime      string        `json:"UpdateTime"`
	UpdateTimestamp string        `json:"UpdateTimestamp"`
	RecordCount     int           `json:"RecordCount"`
	CreateTimestamp int64         `json:"CreateTimestamp"`
	BindVpcs        SPvtzBindVpcs `json:"BindVpcs"`
	IsPtr           bool          `json:"IsPtr"`
}

// list return
type sPrivateZones struct {
	Zone []SPrivateZone `json:"Zone"`
}

type SPrivateZones struct {
	RequestID  string        `json:"RequestId"`
	PageSize   int           `json:"PageSize"`
	PageNumber int           `json:"PageNumber"`
	TotalPages int           `json:"TotalPages"`
	TotalItems int           `json:"TotalItems"`
	Zones      sPrivateZones `json:"Zones"`
}

// https://help.aliyun.com/document_detail/66243.html?spm=a2c4g.11186623.6.580.761357982tMV0Q
func (client *SAliyunClient) DescribeZones(pageNumber int, pageSize int) (SPrivateZones, error) {
	sZones := SPrivateZones{}
	params := map[string]string{}
	params["Action"] = "DescribeZones"
	params["PageNumber"] = strconv.Itoa(pageNumber)
	params["PageSize"] = strconv.Itoa(pageSize)
	resp, err := client.pvtzRequest("DescribeZones", params)
	if err != nil {
		return sZones, errors.Wrap(err, "DescribeZones")
	}
	err = resp.Unmarshal(&sZones)
	if err != nil {
		return sZones, errors.Wrap(err, "resp.Unmarshal")
	}
	return sZones, nil
}

// 没有vpc等详细信息
func (client *SAliyunClient) GetAllZones() ([]SPrivateZone, error) {
	pageNumber := 0
	szones := []SPrivateZone{}
	for {
		pageNumber++
		zones, err := client.DescribeZones(pageNumber, 20)
		if err != nil {
			return nil, errors.Wrapf(err, "client.DescribeZones(%d, 20)", pageNumber)
		}
		szones = append(szones, zones.Zones.Zone...)
		if len(szones) >= zones.TotalItems {
			break
		}
	}
	return szones, nil
}

func (client *SAliyunClient) GetAllZonesInfo() ([]SPrivateZone, error) {
	spvtzs := []SPrivateZone{}
	szones, err := client.GetAllZones()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetAllZones()")
	}
	for i := 0; i < len(szones); i++ {
		spvtz, err := client.DescribeZoneInfo(szones[i].ZoneID)
		if err != nil {
			return nil, errors.Wrapf(err, "client.DescribeZoneInfo(%s)", szones[i].ZoneID)
		}
		spvtzs = append(spvtzs, *spvtz)
	}
	return spvtzs, nil
}

func (client *SAliyunClient) GetPrivateICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	izones := []cloudprovider.ICloudDnsZone{}
	szones, err := client.GetAllZonesInfo()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetAllZonesInfo()")
	}
	for i := 0; i < len(szones); i++ {
		izones = append(izones, &szones[i])
	}
	return izones, nil
}

func (client *SAliyunClient) DescribeZoneInfo(zoneId string) (*SPrivateZone, error) {
	params := map[string]string{}
	params["Action"] = "DescribeZoneInfo"
	params["ZoneId"] = zoneId
	resp, err := client.pvtzRequest("DescribeZoneInfo", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeZoneInfo")
	}
	sZone := &SPrivateZone{client: client}
	err = resp.Unmarshal(sZone)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return sZone, nil
}

func (client *SAliyunClient) GetPrivateICloudDnsZoneById(id string) (cloudprovider.ICloudDnsZone, error) {
	pvtzs, err := client.GetAllZones()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetAllZones()")
	}
	index := -1
	for i := 0; i < len(pvtzs); i++ {
		if pvtzs[i].ZoneID == id {
			index = i
			break
		}
	}
	if index < 0 || index >= len(pvtzs) {
		return nil, cloudprovider.ErrNotFound
	}
	izone, err := client.DescribeZoneInfo(id)
	if err != nil {
		return nil, errors.Wrapf(err, "client.DescribeZoneInfo(%s)", id)
	}
	return izone, nil
}

func (client *SAliyunClient) DeleteZone(zoneId string) error {
	params := map[string]string{}
	params["Action"] = "DeleteZone"
	params["ZoneId"] = zoneId
	_, err := client.pvtzRequest("DeleteZone", params)
	if err != nil {
		return errors.Wrap(err, "DeleteZone")
	}
	return nil
}

func (client *SAliyunClient) AddZone(zoneName string) (string, error) {
	params := map[string]string{}
	params["Action"] = "AddZone"
	params["ZoneName"] = zoneName
	ret, err := client.pvtzRequest("AddZone", params)
	if err != nil {
		return "", errors.Wrap(err, "AddZone")
	}
	zoneId := ""
	return zoneId, ret.Unmarshal(&zoneId, "ZoneId")
}

func (client *SAliyunClient) CreateZone(opts *cloudprovider.SDnsZoneCreateOptions) (*SPrivateZone, error) {
	zoneId, err := client.AddZone(opts.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "client.AddZone(%s)", opts.Name)
	}
	err = client.BindZoneVpcs(zoneId, opts.Vpcs)
	if err != nil {
		return nil, errors.Wrapf(err, " client.BindZoneVpcs(%s,%s)", zoneId, jsonutils.Marshal(opts.Vpcs).String())
	}
	return client.DescribeZoneInfo(zoneId)
}

func (client *SAliyunClient) CreatePrivateICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	izone, err := client.CreateZone(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "client.CreateZone(%s)", jsonutils.Marshal(opts).String())
	}
	return izone, nil
}

func (client *SAliyunClient) BindZoneVpc(ZoneId string, vpc *cloudprovider.SPrivateZoneVpc) error {
	params := map[string]string{}
	params["Action"] = "BindZoneVpc"
	params["ZoneId"] = ZoneId
	params["Vpcs.1.RegionId"] = vpc.RegionId
	params["Vpcs.1.VpcId"] = vpc.Id
	_, err := client.pvtzRequest("BindZoneVpc", params)
	if err != nil {
		return errors.Wrap(err, "BindZoneVpc")
	}
	return nil
}

func (client *SAliyunClient) BindZoneVpcs(zoneId string, vpc []cloudprovider.SPrivateZoneVpc) error {
	params := map[string]string{}
	params["Action"] = "BindZoneVpc"
	params["ZoneId"] = zoneId
	index := ""
	for i := 0; i < len(vpc); i++ {
		index = strconv.Itoa(i + 1)
		params["Vpcs."+index+".RegionId"] = vpc[i].RegionId
		params["Vpcs."+index+".VpcId"] = vpc[i].Id
	}
	_, err := client.pvtzRequest("BindZoneVpc", params)
	if err != nil {
		return errors.Wrap(err, "BindZoneVpc")
	}
	return nil
}

func (client *SAliyunClient) UnBindZoneVpcs(zoneId string) error {
	params := map[string]string{}
	params["Action"] = "BindZoneVpc"
	params["ZoneId"] = zoneId
	_, err := client.pvtzRequest("BindZoneVpc", params)
	if err != nil {
		return errors.Wrap(err, "BindZoneVpc")
	}
	return nil
}

func (self *SPrivateZone) GetId() string {
	return self.ZoneID
}

func (self *SPrivateZone) GetName() string {
	return self.ZoneName
}

func (self *SPrivateZone) GetGlobalId() string {
	return self.GetId()
}

func (self *SPrivateZone) GetStatus() string {
	return api.DNS_ZONE_STATUS_AVAILABLE
}

func (self *SPrivateZone) Refresh() error {
	szone, err := self.client.DescribeZoneInfo(self.ZoneID)
	if err != nil {
		return errors.Wrapf(err, "self.client.DescribeZoneInfo(%s)", self.ZoneID)
	}
	return jsonutils.Update(self, szone)
}

func (self *SPrivateZone) GetZoneType() cloudprovider.TDnsZoneType {
	return cloudprovider.PrivateZone
}

func (self *SPrivateZone) GetOptions() *jsonutils.JSONDict {
	return nil
}

func (self *SPrivateZone) GetICloudVpcIds() ([]string, error) {
	var ret []string
	for i := 0; i < len(self.BindVpcs.Vpc); i++ {
		ret = append(ret, self.BindVpcs.Vpc[i].VpcID)
	}
	return ret, nil
}

func (self *SPrivateZone) AddVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	vpcs := []cloudprovider.SPrivateZoneVpc{}
	for i := 0; i < len(self.BindVpcs.Vpc); i++ {
		vpc := cloudprovider.SPrivateZoneVpc{}
		vpc.Id = self.BindVpcs.Vpc[i].VpcID
		vpc.RegionId = self.BindVpcs.Vpc[i].RegionID
		vpcs = append(vpcs, vpc)
	}
	vpcs = append(vpcs, *vpc)
	return self.client.BindZoneVpcs(self.ZoneID, vpcs)
}

func (self *SPrivateZone) RemoveVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	vpcs := []cloudprovider.SPrivateZoneVpc{}
	for i := 0; i < len(self.BindVpcs.Vpc); i++ {
		newVpc := cloudprovider.SPrivateZoneVpc{}
		if self.BindVpcs.Vpc[i].VpcID == vpc.Id && self.BindVpcs.Vpc[i].RegionID == vpc.RegionId {
			continue
		}
		newVpc.Id = self.BindVpcs.Vpc[i].VpcID
		newVpc.RegionId = self.BindVpcs.Vpc[i].RegionID
		vpcs = append(vpcs, newVpc)
	}
	return self.client.BindZoneVpcs(self.ZoneID, vpcs)
}

func (self *SPrivateZone) GetIDnsRecordSets() ([]cloudprovider.ICloudDnsRecordSet, error) {
	zonerecords, err := self.client.GetAllZoneRecords(self.ZoneID)
	if err != nil {
		return nil, errors.Wrapf(err, "self.client.GetAllZoneRecords(%s)", self.ZoneID)
	}
	result := []cloudprovider.ICloudDnsRecordSet{}
	for i := 0; i < len(zonerecords); i++ {
		zonerecords[i].szone = self
		result = append(result, &zonerecords[i])
	}
	return result, nil
}

func (self *SPrivateZone) SyncDnsRecordSets(common, add, del, update []cloudprovider.DnsRecordSet) error {
	for i := 0; i < len(del); i++ {
		err := self.client.DeleteZoneRecord(del[i].ExternalId)
		if err != nil {
			return errors.Wrapf(err, "self.client.DeleteZoneRecord(%s)", del[i].ExternalId)
		}
	}

	for i := 0; i < len(add); i++ {
		recordId, err := self.client.AddZoneRecord(self.ZoneID, add[i])
		if err != nil {
			return errors.Wrapf(err, "self.client.AddZoneRecord(%s,%s)", self.ZoneID, jsonutils.Marshal(add[i]).String())
		}
		if !add[i].Enabled {
			err = self.client.SetZoneRecordStatus(recordId, "DISABLE")
			if err != nil {
				return errors.Wrapf(err, "self.client.SetZoneRecordStatus(%s,%t)", recordId, add[i].Enabled)
			}
		}
	}

	for i := 0; i < len(update); i++ {
		// ENABLE: 启用解析 DISABLE: 暂停解析
		status := "ENABLE"
		if !update[i].Enabled {
			status = "DISABLE"
		}
		err := self.client.SetZoneRecordStatus(update[i].ExternalId, status)
		if err != nil {
			return errors.Wrapf(err, "self.client.SetZoneRecordStatus(%s,%t)", update[i].ExternalId, update[i].Enabled)
		}
		err = self.client.UpdateZoneRecord(update[i])
		if err != nil {
			return errors.Wrapf(err, "self.client.UpdateZoneRecord(%s)", jsonutils.Marshal(update[i]).String())
		}
	}

	return nil
}

func (self *SPrivateZone) Delete() error {
	if len(self.BindVpcs.Vpc) > 0 {
		err := self.client.UnBindZoneVpcs(self.ZoneID)
		if err != nil {
			return errors.Wrapf(err, "self.client.UnBindZoneVpcs(%s)", self.ZoneID)
		}
	}
	return self.client.DeleteZone(self.ZoneID)
}

func (self *SPrivateZone) GetDnsProductType() cloudprovider.TDnsProductType {
	return ""
}

func (self *SPrivateZone) GetProperlyTTL(ttl int64) int64 {
	if ttl < allowedTtls[0] {
		return allowedTtls[0]
	}
	for i := 0; i < len(allowedTtls)-1; i++ {
		if ttl > allowedTtls[i] && ttl < allowedTtls[i+1] {
			return allowedTtls[i]
		}
	}
	return allowedTtls[len(allowedTtls)-1]
}
