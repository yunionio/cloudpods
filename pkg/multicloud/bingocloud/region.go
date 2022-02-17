/*
 * @Author: your name
 * @Date: 2022-02-16 18:07:12
 * @LastEditTime: 2022-02-17 16:08:05
 * @LastEditors: Please set LastEditors
 * @Description: 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 * @FilePath: \cloudpods\pkg\multicloud\bingocloud\region.go
 */
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

package bingocloud

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	client *SBingoCloudClient

	multicloud.SRegionOssBase
	multicloud.SRegionLbBase
	multicloud.SRegionEipBase
	multicloud.SRegionZoneBase
	multicloud.SRegionVpcBase

	RegionId       string
	RegionName     string
	Hypervisor     string
	NetworkMode    string
	RegionEndpoint string
}

func (self *SRegion) GetClient() *SBingoCloudClient {
	return self.client
}

func (self *SRegion) invoke(action string, params map[string]string) (jsonutils.JSONObject, error) {
	return self.client.invoke(action, params)
}

func (self *SBingoCloudClient) GetRegions() ([]SRegion, error) {
	resp, err := self.invoke("DescribeRegions", nil)
	if err != nil {
		return nil, err
	}
	result := struct {
		RegionInfo struct {
			Item []SRegion
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.RegionInfo.Item, nil
}

func (self *SRegion) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) CancelSnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateICloudAccessGroup(opts *cloudprovider.SAccessGroup) (cloudprovider.ICloudAccessGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateICloudFileSystem(opts *cloudprovider.FileSystemCraeteOptions) (cloudprovider.ICloudFileSystem, error) {
	return nil, cloudprovider.ErrNotImplemented
}

/////////////////startMyself
func (self *SRegion) CreateICloudWafInstance(opts *cloudprovider.WafCreateOptions) (cloudprovider.ICloudWafInstance, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateIDBInstance(desc *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateIElasticcaches(ec *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateISku(opts *cloudprovider.SServerSkuCreateOption) (cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateInternetGateway() (cloudprovider.ICloudInternetGateway, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateSnapshotPolicy(*cloudprovider.SnapshotPolicyInput) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SRegion) DeleteSnapshotPolicy(string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetCapabilities() []string {
	return nil
}

func (self *SRegion) GetCloudEnv() string {
	return ""
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	kong := cloudprovider.SGeographicInfo{}
	return kong
}

// GetGlobalId() string //返回IP即可
func (self *SRegion) GetGlobalId() string {
	bingoURL := "http://10.1.33.25:8663/main.yaws"
	return bingoURL
}

func (self *SRegion) GetName() string {
	return self.RegionName
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName())
	return table
}

func (self *SRegion) GetICloudAccessGroupById(id string) (cloudprovider.ICloudAccessGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetICloudAccessGroups() ([]cloudprovider.ICloudAccessGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetICloudAppById(id string) (cloudprovider.ICloudApp, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetICloudApps() ([]cloudprovider.ICloudApp, error) {
	return nil, cloudprovider.ErrNotImplemented
}

//获取公有云操作日志接口
func (self *SRegion) GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]cloudprovider.ICloudEvent, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetICloudFileSystemById(id string) (cloudprovider.ICloudFileSystem, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetICloudFileSystems() ([]cloudprovider.ICloudFileSystem, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetICloudKafkaById(id string) (cloudprovider.ICloudKafka, error) {
	return nil, cloudprovider.ErrNotImplemented
}
