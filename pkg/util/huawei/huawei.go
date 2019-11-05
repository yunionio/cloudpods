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
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/huawei/client"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth/credentials"
)

/*
待解决问题：
1.同步的子账户中有一条空记录.需要查原因
2.安全组同步需要进一步确认
3.实例接口需要进一步确认
4.BGP type 目前是hard code在代码中。需要考虑从cloudmeta服务中查询
*/

const (
	CLOUD_PROVIDER_HUAWEI    = api.CLOUD_PROVIDER_HUAWEI
	CLOUD_PROVIDER_HUAWEI_CN = "华为云"

	HUAWEI_INTERNATIONAL_CLOUDENV = "InternationalCloud"
	HUAWEI_CHINA_CLOUDENV         = "ChinaCloud"

	HUAWEI_DEFAULT_REGION = "cn-north-1"
	HUAWEI_API_VERSION    = "2018-12-25"
)

type SHuaweiClient struct {
	signer auth.Signer

	debug bool

	providerId   string
	providerName string
	projectId    string // 华为云项目ID.
	accessUrl    string // 服务区域 ChinaCloud | InternationalCloud
	accessKey    string
	secret       string
	iregions     []cloudprovider.ICloudRegion
}

// 进行资源操作时参数account 对应数据库cloudprovider表中的account字段,由accessKey和projectID两部分组成，通过"/"分割。
// 初次导入Subaccount时，参数account对应cloudaccounts表中的account字段，即accesskey。此时projectID为空，
// 只能进行同步子账号、查询region列表等projectId无关的操作。
// todo: 通过accessurl支持国际站。目前暂时未支持国际站
func NewHuaweiClient(providerId, providerName, accessurl, accessKey, secret, projectId string, debug bool) (*SHuaweiClient, error) {
	client := SHuaweiClient{
		providerId:   providerId,
		providerName: providerName,
		projectId:    projectId,
		accessUrl:    accessurl,
		accessKey:    accessKey,
		secret:       secret,
		debug:        debug,
	}
	err := client.init()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (self *SHuaweiClient) init() error {
	err := self.fetchRegions()
	if err != nil {
		return err
	}
	err = self.initSigner()
	if err != nil {
		return err
	}
	return nil
}

func (self *SHuaweiClient) initSigner() error {
	var err error
	cred := credentials.NewAccessKeyCredential(self.accessKey, self.accessKey)
	self.signer, err = auth.NewSignerWithCredential(cred)
	if err != nil {
		return err
	}
	return nil
}

func (self *SHuaweiClient) fetchRegions() error {
	huawei, _ := client.NewClientWithAccessKey("", "", self.accessKey, self.secret, self.debug)
	regions := make([]SRegion, 0)
	err := doListAll(huawei.Regions.List, nil, &regions)
	if err != nil {
		return err
	}

	filtedRegions := make([]SRegion, 0)
	if len(self.projectId) > 0 {
		project, err := self.GetProjectById(self.projectId)
		if err != nil {
			return err
		}

		regionId := strings.Split(project.Name, "_")[0]
		for _, region := range regions {
			if region.ID == regionId {
				filtedRegions = append(filtedRegions, region)
			}
		}
	} else {
		filtedRegions = regions
	}

	self.iregions = make([]cloudprovider.ICloudRegion, len(filtedRegions))
	for i := 0; i < len(filtedRegions); i += 1 {
		filtedRegions[i].client = self
		_, err := filtedRegions[i].getECSClient()
		if err != nil {
			return err
		}
		self.iregions[i] = &filtedRegions[i]
	}
	return nil
}

func (self *SHuaweiClient) GetCloudRegionExternalIdPrefix() string {
	if len(self.projectId) > 0 {
		return self.iregions[0].GetGlobalId()
	} else {
		return CLOUD_PROVIDER_HUAWEI
	}
}

func (self *SHuaweiClient) UpdateAccount(accessKey, secret string) error {
	if self.accessKey != accessKey || self.secret != secret {
		self.accessKey = accessKey
		self.secret = secret
		return self.fetchRegions()
	} else {
		return nil
	}
}

func (self *SHuaweiClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SHuaweiClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	projects, err := self.fetchProjects()
	if err != nil {
		return nil, err
	}

	// https://support.huaweicloud.com/api-iam/zh-cn_topic_0074171149.html
	subAccounts := make([]cloudprovider.SSubAccount, 0)
	for i := range projects {
		project := projects[i]
		// name 为MOS的project是华为云内部的一个特殊project。不需要同步到本地
		if strings.ToLower(project.Name) == "mos" {
			continue
		}
		s := cloudprovider.SSubAccount{
			Name:         fmt.Sprintf("%s-%s", self.providerName, project.Name),
			State:        api.CLOUD_PROVIDER_CONNECTED,
			Account:      fmt.Sprintf("%s/%s", self.accessKey, project.ID),
			HealthStatus: project.GetHealthStatus(),
		}

		subAccounts = append(subAccounts, s)
	}

	return subAccounts, nil
}

func (self *SHuaweiClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SHuaweiClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHuaweiClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = HUAWEI_DEFAULT_REGION
	}
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SHuaweiClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
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

func (self *SHuaweiClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ivpc, err := self.iregions[i].GetIVpcById(id)
		if err == nil {
			return ivpc, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHuaweiClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		istorage, err := self.iregions[i].GetIStorageById(id)
		if err == nil {
			return istorage, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

// 总账户余额
type SAccountBalance struct {
	AvailableAmount float64
}

// 账户余额
// https://support.huaweicloud.com/api-oce/zh-cn_topic_0109685133.html
type SBalance struct {
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	AccountID        string  `json:"account_id"`
	AccountType      int64   `json:"account_type"`
	DesignatedAmount *int64  `json:"designated_amount,omitempty"`
	CreditAmount     *int64  `json:"credit_amount,omitempty"`
	MeasureUnit      int64   `json:"measure_unit"`
}

// 这里的余额指的是所有租户的总余额
func (self *SHuaweiClient) QueryAccountBalance() (*SAccountBalance, error) {
	domains, err := self.getEnabledDomains()
	if err != nil {
		return nil, err
	}

	amount := float64(0)
	for _, domain := range domains {
		v, err := self.queryDomainBalance(domain.ID)
		if err != nil {
			return nil, err
		}

		amount += v
	}

	return &SAccountBalance{AvailableAmount: amount}, nil
}

// https://support.huaweicloud.com/api-bpconsole/zh-cn_topic_0075213309.html
func (self *SHuaweiClient) queryDomainBalance(domainId string) (float64, error) {
	huawei, _ := client.NewClientWithAccessKey("", "", self.accessKey, self.secret, self.debug)
	huawei.Balances.SetDomainId(domainId)
	balances := make([]SBalance, 0)
	err := doListAll(huawei.Balances.List, nil, &balances)
	if err != nil {
		return 0, err
	}

	amount := float64(0)
	for _, balance := range balances {
		amount += balance.Amount
	}

	return amount, nil
}

func (self *SHuaweiClient) GetVersion() string {
	return HUAWEI_API_VERSION
}

func (self *SHuaweiClient) GetAccessEnv() string {
	switch self.accessUrl {
	case HUAWEI_INTERNATIONAL_CLOUDENV:
		return api.CLOUD_ACCESS_ENV_HUAWEI_GLOBAL
	case HUAWEI_CHINA_CLOUDENV:
		return api.CLOUD_ACCESS_ENV_HUAWEI_CHINA
	default:
		return api.CLOUD_ACCESS_ENV_HUAWEI_GLOBAL
	}
}
