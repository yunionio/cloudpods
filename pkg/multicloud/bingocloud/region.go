/*
 * @Author: your name
 * @Date: 2022-02-16 18:07:12
 * @LastEditTime: 2022-02-16 22:37:06
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
	"fmt"
	"io"
	"net/http"
	"net/url"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRegion struct {
	client *SBingoCloudClient

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

func (self *SRegion) GetId() string {
	return self.RegionId
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", api.CLOUD_PROVIDER_BINGO_CLOUD, self.RegionId)
}

func (self *SRegion) GetName() string {
	return self.RegionName
}

func (self *SRegion) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(self.GetName()).CN(self.GetName())
	return table
}

func (self *SRegion) CreateEIP(opts *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetISecurityGroupByName(opts *cloudprovider.SecurityGroupFilterOptions) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetCapabilities() []string {
	return self.client.GetCapabilities()
}

func (self *SRegion) GetCloudEnv() string {
	return ""
}

func (self *SRegion) GetProvider() string {
	return api.CLOUD_PROVIDER_BINGO_CLOUD
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	return []cloudprovider.ICloudEIP{}, nil
}

/////////////

// func (self *SRegion) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
// 	return self.CreateVpc(opts)
// }

// func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
// 	vpc, err := self.GetVpc(id)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "GetVpc(%s)", id)
// 	}
// 	return vpc, nil
// }

// func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
// 	vpcs, err := self.GetVpcs()
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "GetVpcs")
// 	}
// 	ret := []cloudprovider.ICloudVpc{}
// 	for i := range vpcs {
// 		vpcs[i].region = self
// 		ret = append(ret, &vpcs[i])
// 	}
// 	return ret, nil
// }

// func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
// 	clusters, err := self.GetClusters()
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "GetClusters")
// 	}
// 	ret := []cloudprovider.ICloudZone{}
// 	for i := range clusters {
// 		ret = append(ret, &SZone{
// 			SCluster: clusters[i],
// 			region:   self,
// 		})
// 	}
// 	return ret, nil
// }

// func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
// 	zones, err := self.GetIZones()
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "GetIZones")
// 	}
// 	for i := range zones {
// 		if zones[i].GetGlobalId() == id {
// 			return zones[i], nil
// 		}
// 	}
// 	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
// }

// func (self *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
// 	zones, err := self.GetIZones()
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "GetIZones")
// 	}
// 	ret := []cloudprovider.ICloudHost{}
// 	for i := range zones {
// 		part, err := zones[i].GetIHosts()
// 		if err != nil {
// 			return nil, errors.Wrapf(err, "GetIHost")
// 		}
// 		ret = append(ret, part...)
// 	}
// 	return ret, nil
// }

// func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
// 	vm, err := self.GetInstance(id)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return vm, nil
// }

// func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
// 	hosts, err := self.GetIHosts()
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "GetIHosts")
// 	}
// 	for i := range hosts {
// 		if hosts[i].GetGlobalId() == id {
// 			return hosts[i], nil
// 		}
// 	}
// 	return nil, cloudprovider.ErrNotFound
// }

//////////////

func (self *SRegion) list(res string, params url.Values, retVal interface{}) (int, error) {
	return self.client.list(res, params, retVal)
}

func (self *SRegion) get(res, id string, params url.Values, retVal interface{}) error {
	return self.client.get(res, id, params, retVal)
}

func (self *SRegion) listAll(res string, params url.Values, retVal interface{}) error {
	return self.client.listAll(res, params, retVal)
}

func (self *SRegion) post(res string, body jsonutils.JSONObject, retVal interface{}) error {
	return self.client.post(res, body, retVal)
}

func (self *SRegion) delete(res string, id string) error {
	return self.client.delete(res, id)
}

func (self *SRegion) update(res string, id string, body jsonutils.JSONObject, retVal interface{}) error {
	return self.client.update(res, id, body, retVal)
}

func (self *SRegion) upload(res string, id string, header http.Header, body io.Reader) (jsonutils.JSONObject, error) {
	return self.client.upload(res, id, header, body)
}

func (self *SRegion) getTask(id string) (*STask, error) {
	task := &STask{}
	return task, self.get("tasks", id, nil, task)
}
