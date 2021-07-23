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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/auth/credentials"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/obs"
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
	CLOUD_PROVIDER_HUAWEI_EN = "Huawei"

	HUAWEI_INTERNATIONAL_CLOUDENV = "InternationalCloud"
	HUAWEI_CHINA_CLOUDENV         = "ChinaCloud"

	HUAWEI_DEFAULT_REGION = "cn-north-1"
	HUAWEI_API_VERSION    = "2018-12-25"
)

var HUAWEI_REGION_CACHES = map[string]userRegionsCache{}

type userRegionsCache struct {
	UserId   string
	ExpireAt time.Time
	Regions  []SRegion
}

type HuaweiClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	projectId    string // 华为云项目ID.
	cloudEnv     string // 服务区域 ChinaCloud | InternationalCloud
	accessKey    string
	accessSecret string

	debug bool
}

func NewHuaweiClientConfig(cloudEnv, accessKey, accessSecret, projectId string) *HuaweiClientConfig {
	cfg := &HuaweiClientConfig{
		projectId:    projectId,
		cloudEnv:     cloudEnv,
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
	return cfg
}

func (cfg *HuaweiClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *HuaweiClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *HuaweiClientConfig) Debug(debug bool) *HuaweiClientConfig {
	cfg.debug = debug
	return cfg
}

type SHuaweiClient struct {
	*HuaweiClientConfig

	signer auth.Signer

	isMainProject bool // whether the project is the main project in the region

	ownerId         string
	ownerName       string
	ownerCreateTime time.Time

	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket

	projects []SProject
	regions  []SRegion
}

// 进行资源操作时参数account 对应数据库cloudprovider表中的account字段,由accessKey和projectID两部分组成，通过"/"分割。
// 初次导入Subaccount时，参数account对应cloudaccounts表中的account字段，即accesskey。此时projectID为空，
// 只能进行同步子账号、查询region列表等projectId无关的操作。
// todo: 通过accessurl支持国际站。目前暂时未支持国际站
func NewHuaweiClient(cfg *HuaweiClientConfig) (*SHuaweiClient, error) {
	client := SHuaweiClient{
		HuaweiClientConfig: cfg,
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
		return errors.Wrap(err, "initSigner")
	}
	err = self.initOwner()
	if err != nil {
		return errors.Wrap(err, "fetchOwner")
	}
	if self.debug {
		log.Debugf("OwnerId: %s name: %s", self.ownerId, self.ownerName)
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

func (self *SHuaweiClient) newRegionAPIClient(regionId string) (*client.Client, error) {
	cli, err := client.NewClientWithAccessKey(regionId, self.ownerId, self.projectId, self.accessKey, self.accessSecret, self.debug)
	if err != nil {
		return nil, err
	}

	httpClient := self.cpcfg.AdaptiveTimeoutHttpClient()
	cli.SetHttpClient(httpClient)

	return cli, nil
}

func (self *SHuaweiClient) newGeneralAPIClient() (*client.Client, error) {
	cli, err := client.NewClientWithAccessKey("", self.ownerId, "", self.accessKey, self.accessSecret, self.debug)
	if err != nil {
		return nil, err
	}

	httpClient := self.cpcfg.AdaptiveTimeoutHttpClient()
	cli.SetHttpClient(httpClient)

	return cli, nil
}

func (self *SHuaweiClient) fetchRegions() error {
	huawei, _ := self.newGeneralAPIClient()
	if self.regions == nil {
		userId, err := self.GetUserId()
		if err != nil {
			return errors.Wrap(err, "GetUserId")
		}

		if regionsCache, ok := HUAWEI_REGION_CACHES[userId]; !ok || regionsCache.ExpireAt.Sub(time.Now()).Seconds() > 0 {
			regions := make([]SRegion, 0)
			err := doListAll(huawei.Regions.List, nil, &regions)
			if err != nil {
				return errors.Wrap(err, "Regions.List")
			}

			HUAWEI_REGION_CACHES[userId] = userRegionsCache{ExpireAt: time.Now().Add(24 * time.Hour), UserId: userId, Regions: regions}
		}

		self.regions = HUAWEI_REGION_CACHES[userId].Regions
	}

	filtedRegions := make([]SRegion, 0)
	if len(self.projectId) > 0 {
		project, err := self.GetProjectById(self.projectId)
		if err != nil {
			return err
		}

		regionId := strings.Split(project.Name, "_")[0]
		for _, region := range self.regions {
			if region.ID == regionId {
				filtedRegions = append(filtedRegions, region)
			}
		}
		if regionId == project.Name {
			self.isMainProject = true
		}
	} else {
		filtedRegions = self.regions
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

func (self *SHuaweiClient) invalidateIBuckets() {
	self.iBuckets = nil
}

func (self *SHuaweiClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if self.iBuckets == nil {
		err := self.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return self.iBuckets, nil
}

func getOBSEndpoint(regionId string) string {
	return fmt.Sprintf("obs.%s.myhuaweicloud.com", regionId)
}

func (client *SHuaweiClient) getOBSClient(regionId string) (*obs.ObsClient, error) {
	endpoint := getOBSEndpoint(regionId)
	return obs.New(client.accessKey, client.accessSecret, endpoint)
}

func (self *SHuaweiClient) fetchBuckets() error {
	obscli, err := self.getOBSClient(HUAWEI_DEFAULT_REGION)
	if err != nil {
		return errors.Wrap(err, "getOBSClient")
	}
	input := &obs.ListBucketsInput{QueryLocation: true}
	output, err := obscli.ListBuckets(input)
	if err != nil {
		return errors.Wrap(err, "obscli.ListBuckets")
	}
	self.ownerId = output.Owner.ID

	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range output.Buckets {
		bInfo := output.Buckets[i]
		region, err := self.getIRegionByRegionId(bInfo.Location)
		if err != nil {
			log.Errorf("fail to find region %s", bInfo.Location)
			continue
		}
		b := SBucket{
			region: region.(*SRegion),

			Name:         bInfo.Name,
			Location:     bInfo.Location,
			CreationDate: bInfo.CreationDate,
		}
		ret = append(ret, &b)
	}
	self.iBuckets = ret
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
	if self.accessKey != accessKey || self.accessSecret != secret {
		self.accessKey = accessKey
		self.accessSecret = secret
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
		// https://www.huaweicloud.com/notice/2018/20190618171312411.html
		expiredAt, _ := timeutils.ParseTimeStr("2020-09-16 00:00:00")
		if !self.ownerCreateTime.IsZero() && self.ownerCreateTime.After(expiredAt) && strings.ToLower(project.Name) == "cn-north-1" {
			continue
		}
		s := cloudprovider.SSubAccount{
			Name:         fmt.Sprintf("%s-%s", self.cpcfg.Name, project.Name),
			Account:      fmt.Sprintf("%s/%s", self.accessKey, project.ID),
			HealthStatus: project.GetHealthStatus(),
		}

		subAccounts = append(subAccounts, s)
	}

	return subAccounts, nil
}

func (client *SHuaweiClient) GetAccountId() string {
	return client.ownerId
}

func (client *SHuaweiClient) GetIamLoginUrl() string {
	return fmt.Sprintf("https://auth.huaweicloud.com/authui/login.html?account=%s#/login", client.ownerName)
}

func (self *SHuaweiClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SHuaweiClient) getIRegionByRegionId(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		log.Debugf("%d ID: %s", i, self.iregions[i].GetId())
		if self.iregions[i].GetId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
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
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
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
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
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
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

// 总账户余额
type SAccountBalance struct {
	AvailableAmount  float64
	CreditAmount     float64
	DesignatedAmount float64
}

// 账户余额
// https://support.huaweicloud.com/api-oce/zh-cn_topic_0109685133.html
type SBalance struct {
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	AccountID        string  `json:"account_id"`
	AccountType      int64   `json:"account_type"`
	DesignatedAmount float64 `json:"designated_amount,omitempty"`
	CreditAmount     float64 `json:"credit_amount,omitempty"`
	MeasureUnit      int64   `json:"measure_unit"`
}

// 这里的余额指的是所有租户的总余额
func (self *SHuaweiClient) QueryAccountBalance() (*SAccountBalance, error) {
	domains, err := self.getEnabledDomains()
	if err != nil {
		return nil, err
	}

	result := &SAccountBalance{}
	for _, domain := range domains {
		balances, err := self.queryDomainBalances(domain.ID)
		if err != nil {
			return nil, err
		}
		for _, balance := range balances {
			result.AvailableAmount += balance.Amount
			result.CreditAmount += balance.CreditAmount
			result.DesignatedAmount += balance.DesignatedAmount
		}
	}

	return result, nil
}

// https://support.huaweicloud.com/api-bpconsole/zh-cn_topic_0075213309.html
func (self *SHuaweiClient) queryDomainBalances(domainId string) ([]SBalance, error) {
	huawei, _ := self.newGeneralAPIClient()
	huawei.Balances.SetDomainId(domainId)
	balances := make([]SBalance, 0)
	err := doListAll(huawei.Balances.List, nil, &balances)
	if err != nil {
		return nil, err
	}

	return balances, nil
}

func (self *SHuaweiClient) GetVersion() string {
	return HUAWEI_API_VERSION
}

func (self *SHuaweiClient) GetAccessEnv() string {
	switch self.cloudEnv {
	case HUAWEI_INTERNATIONAL_CLOUDENV:
		return api.CLOUD_ACCESS_ENV_HUAWEI_GLOBAL
	case HUAWEI_CHINA_CLOUDENV:
		return api.CLOUD_ACCESS_ENV_HUAWEI_CHINA
	default:
		return api.CLOUD_ACCESS_ENV_HUAWEI_GLOBAL
	}
}

func (self *SHuaweiClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_SAML_AUTH,
		cloudprovider.CLOUD_CAPABILITY_NAT,
		cloudprovider.CLOUD_CAPABILITY_NAS,
	}
	// huawei objectstore is shared across projects(subscriptions)
	// to avoid multiple project access the same bucket
	// only main project is allow to access objectstore bucket
	if self.isMainProject {
		caps = append(caps, cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE)
	}
	return caps
}

func (self *SHuaweiClient) GetUserId() (string, error) {
	client, err := self.newGeneralAPIClient()
	if err != nil {
		return "", errors.Wrap(err, "SHuaweiClient.GetUserId.newGeneralAPIClient")
	}

	type cred struct {
		UserId string `json:"user_id"`
	}

	ret := &cred{}
	err = DoGet(client.Credentials.Get, self.accessKey, nil, ret)
	if err != nil {
		return "", errors.Wrap(err, "SHuaweiClient.GetUserId.DoGet")
	}

	return ret.UserId, nil
}

// owner id == domain_id == account id
func (self *SHuaweiClient) GetOwnerId() (string, error) {
	userId, err := self.GetUserId()
	if err != nil {
		return "", errors.Wrap(err, "SHuaweiClient.GetOwnerId.GetUserId")
	}

	client, err := self.newGeneralAPIClient()
	if err != nil {
		return "", errors.Wrap(err, "SHuaweiClient.GetOwnerId.newGeneralAPIClient")
	}

	type user struct {
		DomainId   string `json:"domain_id"`
		Name       string `json:"name"`
		CreateTime string
	}

	ret := &user{}
	err = DoGet(client.Users.Get, userId, nil, ret)
	if err != nil {
		return "", errors.Wrap(err, "SHuaweiClient.GetOwnerId.DoGet")
	}
	self.ownerName = ret.Name
	// 2021-02-02 02:43:28.0
	self.ownerCreateTime, _ = timeutils.ParseTimeStr(strings.TrimSuffix(ret.CreateTime, ".0"))
	return ret.DomainId, nil
}

func (self *SHuaweiClient) initOwner() error {
	ownerId, err := self.GetOwnerId()
	if err != nil {
		return errors.Wrap(err, "SHuaweiClient.initOwner")
	}

	self.ownerId = ownerId
	return nil
}
