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

package ucloud

import (
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

/*
UCLOUD 项目：https://docs.ucloud.cn/management_monitor/uproject/projects
项目可认为是云账户下承载资源的容器，当您注册一个UCloud云账户后，系统会默认创建一个项目，您属于的资源都落在此项目下。如您有新的业务要使用云服务，可创建一个新项目，并将新业务部署在新项目下，实现业务之间的网络与逻辑隔离。

1、项目之间默认网络与逻辑隔离，即项目A的主机无法绑定项目B的EIP，默认也无法与项目B的主机内网通信。但联通项目后，uhost、udb、umem可实现内网通信。

2、资源不能在项目间迁移，即项目A内的主机无法迁移至项目B，因其不在一个基础网络内，且逻辑上也是隔离的。但诸如自主镜像等静态资源，您可以提交工单申请迁移至其他项目。

3、只有云账户本身，才能删除项目，且必须是项目被没有资源、没有任何子成员、未与其他项目联通的情况下才可删除。


UCloud DiskType貌似也是一个奇葩的存在
// https://docs.ucloud.cn/api/uhost-api/disk_type
1.在主机创建查询接口中 DISK type 对应 CLOUD_SSD|CLOUD_NORMAL|...
2.在数据盘创建中对应  DataDisk|SSDDataDisk
3.在数据盘查询接口请求中对应   DataDisk|SystemDisk 。在结果中对应DataDisk|SSDDataDisk|SSDSystemDisk|SystemDisk

目前存在的问题：
1.很多国外区域都需要单独申请开通权限才能使用。onecloud有可能调度到未开通权限区域导致失败。
*/

const (
	CLOUD_PROVIDER_UCLOUD    = api.CLOUD_PROVIDER_UCLOUD
	CLOUD_PROVIDER_UCLOUD_CN = "UCloud"

	UCLOUD_DEFAULT_REGION = "cn-bj2"

	UCLOUD_API_VERSION = "2019-02-28"
)

type SUcloudClient struct {
	providerId      string
	providerName    string
	accessKeyId     string
	accessKeySecret string
	projectId       string

	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket

	httpClient *http.Client
	Debug      bool
}

// 进行资源操作时参数account 对应数据库cloudprovider表中的account字段,由accessKey和projectID两部分组成，通过"/"分割。
// 初次导入Subaccount时，参数account对应cloudaccounts表中的account字段，即accesskey。此时projectID为空，只能进行同步子账号（项目）、查询region列表等projectId无关的操作。
func NewUcloudClient(providerId string, providerName string, accessKey string, secret string, projectId string, isDebug bool) (*SUcloudClient, error) {
	client := SUcloudClient{
		providerId:      providerId,
		providerName:    providerName,
		accessKeyId:     accessKey,
		accessKeySecret: secret,
		projectId:       projectId,
		Debug:           isDebug,
	}

	err := client.fetchRegions()
	if err != nil {
		return nil, err
	}

	err = client.fetchBuckets()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (self *SUcloudClient) UpdateAccount(accessKey, secret string) error {
	if self.accessKeyId != accessKey || self.accessKeySecret != secret {
		self.accessKeyId = accessKey
		self.accessKeySecret = secret
		return self.fetchRegions()
	} else {
		return nil
	}
}

func (self *SUcloudClient) commonParams(params SParams, action string) (string, SParams) {
	resultKey, exists := UCLOUD_API_RESULT_KEYS[action]
	if !exists || len(resultKey) == 0 {
		// default key for describe actions
		if strings.HasPrefix(action, "Describe") {
			resultKey = "DataSet"
		}
	}

	if len(self.projectId) > 0 {
		params.Set("ProjectId", self.projectId)
	}
	params.Set("PublicKey", self.accessKeyId)

	return resultKey, params
}

func (self *SUcloudClient) DoListAll(action string, params SParams, result interface{}) error {
	resultKey, params := self.commonParams(params, action)
	return DoListAll(self, action, params, resultKey, result)
}

func (self *SUcloudClient) DoListPart(action string, limit int, offset int, params SParams, result interface{}) (int, int, error) {
	resultKey, params := self.commonParams(params, action)
	params.SetPagination(limit, offset)
	return doListPart(self, action, params, resultKey, result)
}

func (self *SUcloudClient) DoAction(action string, params SParams, result interface{}) error {
	resultKey, params := self.commonParams(params, action)
	err := DoAction(self, action, params, resultKey, result)
	if err != nil {
		return err
	}

	return nil
}

func (self *SUcloudClient) fetchRegions() error {
	type Region struct {
		RegionID   int64  `json:"RegionId"`
		RegionName string `json:"RegionName"`
		IsDefault  bool   `json:"IsDefault"`
		BitMaps    string `json:"BitMaps"`
		Region     string `json:"Region"`
		Zone       string `json:"Zone"`
	}

	params := NewUcloudParams()
	regions := make([]Region, 0)
	err := self.DoListAll("GetRegion", params, &regions)
	if err != nil {
		return err
	}

	regionSet := make(map[string]string, 0)
	for _, region := range regions {
		regionSet[region.Region] = region.Region
	}

	sregions := make([]SRegion, len(regionSet))
	self.iregions = make([]cloudprovider.ICloudRegion, len(regionSet))
	i := 0
	for regionId := range regionSet {
		sregions[i].client = self
		sregions[i].RegionID = regionId
		self.iregions[i] = &sregions[i]
		i += 1
	}

	return nil
}

func (client *SUcloudClient) invalidateIBuckets() {
	client.iBuckets = nil
}

func (client *SUcloudClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if client.iBuckets == nil {
		err := client.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return client.iBuckets, nil
}

func (client *SUcloudClient) fetchBuckets() error {
	buckets := make([]SBucket, 0)
	offset := 0
	limit := 50
	for {
		parts, err := client.listBuckets("", offset, limit)
		if err != nil {
			return errors.Wrap(err, "client.listBuckets")
		}
		if len(parts) > 0 {
			buckets = append(buckets, parts...)
		}
		if len(parts) < limit {
			break
		} else {
			offset += limit
		}
	}
	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range buckets {
		region, err := client.getIRegionByRegionId(buckets[i].Region)
		if err != nil {
			log.Errorf("fail to find iregion %s", buckets[i].Region)
			continue
		}
		buckets[i].region = region.(*SRegion)
		ret = append(ret, &buckets[i])
	}

	client.iBuckets = ret

	return nil
}

func (self *SUcloudClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SUcloudClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	projects, err := self.FetchProjects()
	if err != nil {
		return nil, err
	}

	subAccounts := make([]cloudprovider.SSubAccount, 0)
	for _, project := range projects {
		subAccount := cloudprovider.SSubAccount{}
		subAccount.Name = fmt.Sprintf("%s-%s", self.providerName, project.ProjectName)
		// ucloud账号ID中可能包含/。因此使用::作为分割符号
		subAccount.Account = fmt.Sprintf("%s::%s", self.accessKeyId, project.ProjectID)
		subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL

		subAccounts = append(subAccounts, subAccount)
	}

	return subAccounts, nil
}

func (self *SUcloudClient) GetAccountId() string {
	return "" // no account ID found for ucloud
}

func (self *SUcloudClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func removeDigit(idstr string) string {
	for len(idstr) > 0 && idstr[len(idstr)-1] >= '0' && idstr[len(idstr)-1] <= '9' {
		idstr = idstr[:len(idstr)-1]
	}
	return idstr
}

func (self *SUcloudClient) getIRegionByRegionId(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == id {
			return self.iregions[i], nil
		}
	}
	// retry
	for i := 0; i < len(self.iregions); i += 1 {
		rid := removeDigit(self.iregions[i].GetId())
		rid2 := removeDigit(id)
		if rid == rid2 {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SUcloudClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SUcloudClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = UCLOUD_DEFAULT_REGION
	}
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SUcloudClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SUcloudClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIVpcById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SUcloudClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIStorageById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SUcloudClient) GetCapabilities() []string {
	caps := []string{
		// cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
	}
	return caps
}
