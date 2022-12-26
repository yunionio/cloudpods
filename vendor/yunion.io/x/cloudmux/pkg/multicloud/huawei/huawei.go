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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go/auth/aksk"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/timeutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/auth"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/client/auth/credentials"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/obs"
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

var HUAWEI_REGION_CACHES sync.Map

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
	clientRegion  string

	userId          string
	ownerId         string
	ownerName       string
	ownerCreateTime time.Time

	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket

	projects []SProject
	regions  []SRegion

	httpClient *http.Client
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

func (self *SHuaweiClient) getDefaultClient() *http.Client {
	if self.httpClient != nil {
		return self.httpClient
	}
	self.httpClient = self.cpcfg.AdaptiveTimeoutHttpClient()
	ts, _ := self.httpClient.Transport.(*http.Transport)
	self.httpClient.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response), error) {
		service, method, path := strings.Split(req.URL.Host, ".")[0], req.Method, req.URL.Path
		respCheck := func(resp *http.Response) {
			if resp.StatusCode == 403 {
				if self.cpcfg.UpdatePermission != nil {
					self.cpcfg.UpdatePermission(service, fmt.Sprintf("%s %s", method, path))
				}
			}
		}
		if self.cpcfg.ReadOnly {
			if req.Method == "GET" {
				return respCheck, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return respCheck, nil
	})
	return self.httpClient
}

func (self *SHuaweiClient) newRegionAPIClient(regionId string) (*client.Client, error) {
	projectId := self.projectId
	if len(regionId) == 0 {
		projectId = ""
	}
	cli, err := client.NewPublicCloudClientWithAccessKey(regionId, self.ownerId, projectId, self.accessKey, self.accessSecret, self.debug)
	if err != nil {
		return nil, err
	}

	httpClient := self.getDefaultClient()
	cli.SetHttpClient(httpClient)

	return cli, nil
}

type sPageInfo struct {
	NextMarker string
}

func (self *SHuaweiClient) newGeneralAPIClient() (*client.Client, error) {
	return self.newRegionAPIClient("")
}

func (self *SHuaweiClient) lbList(regionId, resource string, query url.Values) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("https://elb.%s.myhuaweicloud.com/v2/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.GET, url, query, nil)
}

func (self *SHuaweiClient) monitorList(resource string, query url.Values) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("https://ces.%s.myhuaweicloud.com/v1.0/%s/%s", self.clientRegion, self.projectId, resource)
	return self.request(httputils.GET, url, query, nil)
}

func (self *SHuaweiClient) monitorPost(resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("https://ces.%s.myhuaweicloud.com/V1.0/%s/%s", self.clientRegion, self.projectId, resource)
	return self.request(httputils.POST, url, nil, params)
}

func (self *SHuaweiClient) modelartsPoolNetworkList(resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v1/%s/networks", self.clientRegion, self.projectId)
	return self.request(httputils.GET, uri, url.Values{}, params)
}

func (self *SHuaweiClient) modelartsPoolNetworkCreate(params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v1/%s/networks", self.clientRegion, self.projectId)
	return self.request(httputils.POST, uri, url.Values{}, params)
}

func (cli *SHuaweiClient) modelartsPoolNetworkDetail(networkName string) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v1/%s/networks/%s", cli.clientRegion, cli.projectId, networkName)
	return cli.request(httputils.GET, uri, url.Values{}, nil)
}

func (self *SHuaweiClient) modelartsPoolById(poolName string) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v2/%s/pools/%s", self.clientRegion, self.projectId, poolName)
	return self.request(httputils.GET, uri, url.Values{}, nil)
}

func (self *SHuaweiClient) modelartsPoolList(resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v2/%s/%s", self.clientRegion, self.projectId, resource)
	return self.request(httputils.GET, uri, url.Values{}, params)
}

func (cli *SHuaweiClient) modelartsPoolListWithStatus(resource, status string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v2/%s/%s", cli.clientRegion, cli.projectId, resource)
	value := url.Values{}
	value.Add("status", status)
	return cli.request(httputils.GET, uri, value, params)
}

func (self *SHuaweiClient) modelartsPoolCreate(resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v2/%s/%s", self.clientRegion, self.projectId, resource)
	return self.request(httputils.POST, uri, url.Values{}, params)
}

func (self *SHuaweiClient) modelartsPoolDelete(resource, poolName string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v2/%s/pools/%s", self.clientRegion, self.projectId, poolName)
	return self.request(httputils.DELETE, uri, url.Values{}, params)
}

func (self *SHuaweiClient) modelartsPoolUpdate(poolName string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v2/%s/pools/%s", self.clientRegion, self.projectId, poolName)
	urlValue := url.Values{}
	urlValue.Add("time_range", "")
	urlValue.Add("statistics", "")
	urlValue.Add("period", "")
	return self.patchRequest(httputils.PATCH, uri, urlValue, params)
}

func (self *SHuaweiClient) modelartsPoolMonitor(poolName string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v2/%s/pools/%s/monitor", self.clientRegion, self.projectId, poolName)
	return self.request(httputils.GET, uri, url.Values{}, params)
}

func (self *SHuaweiClient) modelartsResourceflavors(resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v1/%s/%s", self.clientRegion, self.projectId, resource)
	return self.request(httputils.GET, uri, url.Values{}, params)
}

func (self *SHuaweiClient) getAKSKList(userId string) (jsonutils.JSONObject, error) {
	params := url.Values{}
	params.Set("user_id", userId)
	uri := fmt.Sprintf("https://iam.cn-north-4.myhuaweicloud.com/v3.0/OS-CREDENTIAL/credentials")
	return self.request(httputils.GET, uri, params, nil)
}

func (self *SHuaweiClient) deleteAKSK(accesskey string) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://iam.cn-north-4.myhuaweicloud.com/v3.0/OS-CREDENTIAL/credentials/%s", accesskey)
	return self.request(httputils.DELETE, uri, url.Values{}, nil)
}

func (self *SHuaweiClient) createAKSK(params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://iam.cn-north-4.myhuaweicloud.com/v3.0/OS-CREDENTIAL/credentials")
	return self.request(httputils.POST, uri, url.Values{}, params)
}

func (self *SHuaweiClient) lbGet(regionId, resource string) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://elb.%s.myhuaweicloud.com/v2/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.GET, uri, url.Values{}, nil)
}

func (self *SHuaweiClient) lbCreate(regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://elb.%s.myhuaweicloud.com/v2/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.POST, uri, url.Values{}, params)
}

func (self *SHuaweiClient) lbUpdate(regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://elb.%s.myhuaweicloud.com/v2/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.PUT, uri, url.Values{}, params)
}

func (self *SHuaweiClient) lbDelete(regionId, resource string) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://elb.%s.myhuaweicloud.com/v2/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.DELETE, uri, url.Values{}, nil)
}

func (self *SHuaweiClient) vpcList(regionId, resource string, query url.Values) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("https://vpc.%s.myhuaweicloud.com/v1/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.GET, url, query, nil)
}

func (self *SHuaweiClient) vpcCreate(regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://vpc.%s.myhuaweicloud.com/v1/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.POST, uri, url.Values{}, params)
}

func (self *SHuaweiClient) vpcGet(regionId, resource string) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://vpc.%s.myhuaweicloud.com/v1/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.GET, uri, url.Values{}, nil)
}

func (self *SHuaweiClient) vpcDelete(regionId, resource string) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://vpc.%s.myhuaweicloud.com/v1/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.DELETE, uri, url.Values{}, nil)
}

func (self *SHuaweiClient) vpcUpdate(regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("https://vpc.%s.myhuaweicloud.com/v1/%s/%s", regionId, self.projectId, resource)
	return self.request(httputils.PUT, uri, url.Values{}, params)
}

type akClient struct {
	client *http.Client
	aksk   aksk.SignOptions
}

func (self *akClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Del("Accept")
	if req.Method == string(httputils.GET) || req.Method == string(httputils.DELETE) || req.Method == string(httputils.PATCH) {
		req.Header.Del("Content-Length")
	}
	aksk.Sign(req, self.aksk)
	return self.client.Do(req)
}

func (self *SHuaweiClient) getAkClient() *akClient {
	return &akClient{
		client: self.getDefaultClient(),
		aksk: aksk.SignOptions{
			AccessKey: self.accessKey,
			SecretKey: self.accessSecret,
		},
	}
}

func (self *SHuaweiClient) request(method httputils.THttpMethod, url string, query url.Values, params map[string]interface{}) (jsonutils.JSONObject, error) {
	client := self.getAkClient()
	if len(query) > 0 {
		url = fmt.Sprintf("%s?%s", url, query.Encode())
	}
	var body jsonutils.JSONObject = nil
	if len(params) > 0 {
		body = jsonutils.Marshal(params)
	}
	header := http.Header{}
	if len(self.projectId) > 0 {
		header.Set("X-Project-Id", self.projectId)
	}
	if strings.Contains(url, "/OS-CREDENTIAL/credentials") && len(self.ownerId) > 0 {
		header.Set("X-Domain-Id", self.ownerId)
	}
	_, resp, err := httputils.JSONRequest(client, context.Background(), method, url, header, body, self.debug)
	if err != nil {
		if e, ok := err.(*httputils.JSONClientError); ok && e.Code == 404 {
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
		}
		return nil, err
	}
	return resp, err
}

func (self *SHuaweiClient) fetchRegions() error {
	huawei, _ := self.newGeneralAPIClient()
	if self.regions == nil {
		userId, err := self.GetUserId()
		if err != nil {
			return errors.Wrap(err, "GetUserId")
		}

		if regionsCache, ok := HUAWEI_REGION_CACHES.Load(userId); !ok || regionsCache.(*userRegionsCache).ExpireAt.Sub(time.Now()).Seconds() > 0 {
			regions := make([]SRegion, 0)
			err := doListAll(huawei.Regions.List, nil, &regions)
			if err != nil {
				return errors.Wrap(err, "Regions.List")
			}

			HUAWEI_REGION_CACHES.Store(userId, &userRegionsCache{ExpireAt: time.Now().Add(24 * time.Hour), UserId: userId, Regions: regions})
		}

		if regionsCache, ok := HUAWEI_REGION_CACHES.Load(userId); ok {
			self.regions = regionsCache.(*userRegionsCache).Regions
		}
	}

	filtedRegions := make([]SRegion, 0)
	if len(self.projectId) > 0 {
		project, err := self.GetProjectById(self.projectId)
		if err != nil {
			return err
		}

		for _, region := range self.regions {
			if strings.Count(project.Name, region.ID) >= 1 {
				self.clientRegion = region.ID
				filtedRegions = append(filtedRegions, region)
			}
			if project.Name == region.ID {
				self.isMainProject = true
			}
		}
	} else {
		filtedRegions = self.regions
	}

	if len(filtedRegions) == 0 {
		return errors.Wrapf(cloudprovider.ErrNotFound, "empty regions")
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

func (self *SHuaweiClient) getOBSClient(regionId string) (*obs.ObsClient, error) {
	endpoint := getOBSEndpoint(regionId)
	cli, err := obs.New(self.accessKey, self.accessSecret, endpoint)
	if err != nil {
		return nil, err
	}
	client := cli.GetClient()
	ts, _ := client.Transport.(*http.Transport)
	client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response), error) {
		method, path := req.Method, req.URL.Path
		respCheck := func(resp *http.Response) {
			if resp.StatusCode == 403 {
				if self.cpcfg.UpdatePermission != nil {
					self.cpcfg.UpdatePermission("obs", fmt.Sprintf("%s %s", method, path))
				}
			}
		}
		if self.cpcfg.ReadOnly {
			if req.Method == "GET" || req.Method == "HEAD" {
				return respCheck, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return respCheck, nil
	})

	return cli, nil
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
		// expiredAt, _ := timeutils.ParseTimeStr("2020-09-16 00:00:00")
		// if !self.ownerCreateTime.IsZero() && self.ownerCreateTime.After(expiredAt) && strings.ToLower(project.Name) == "cn-north-1" {
		// 	continue
		// }
		s := cloudprovider.SSubAccount{
			Name:             fmt.Sprintf("%s-%s", self.cpcfg.Name, project.Name),
			Account:          fmt.Sprintf("%s/%s", self.accessKey, project.ID),
			HealthStatus:     project.GetHealthStatus(),
			DefaultProjectId: "0",
		}
		for j := range self.iregions {
			region := self.iregions[j].(*SRegion)
			if strings.Contains(project.Name, region.ID) {
				s.Desc = region.Locales.ZhCN
				break
			}
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
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_SAML_AUTH,
		cloudprovider.CLOUD_CAPABILITY_NAT,
		cloudprovider.CLOUD_CAPABILITY_NAS,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_MODELARTES,
		cloudprovider.CLOUD_CAPABILITY_VPC_PEER,
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
	if len(self.userId) > 0 {
		return self.userId, nil
	}
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
	self.userId = ret.UserId
	return self.userId, nil
}

// owner id == domain_id == account id
func (self *SHuaweiClient) GetOwnerId() (string, error) {
	if len(self.ownerId) > 0 {
		return self.ownerId, nil
	}
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
	self.ownerId = ret.DomainId
	return self.ownerId, nil
}

func (self *SHuaweiClient) initOwner() error {
	_, err := self.GetOwnerId()
	if err != nil {
		return errors.Wrap(err, "SHuaweiClient.initOwner")
	}
	return nil
}

func (self *SHuaweiClient) patchRequest(method httputils.THttpMethod, url string, query url.Values, params map[string]interface{}) (jsonutils.JSONObject, error) {
	client := self.getAkClient()
	if len(query) > 0 {
		url = fmt.Sprintf("%s?%s", url, query.Encode())
	}
	var body jsonutils.JSONObject = nil
	if len(params) > 0 {
		body = jsonutils.Marshal(params)
	}
	header := http.Header{}
	if len(self.projectId) > 0 {
		header.Set("X-Project-Id", self.projectId)
	}
	var bodystr string
	if !gotypes.IsNil(body) {
		bodystr = body.String()
	}
	jbody := strings.NewReader(bodystr)
	header.Set("Content-Length", strconv.FormatInt(int64(len(bodystr)), 10))
	header.Set("Content-Type", "application/merge-patch+json")
	resp, err := httputils.Request(client, context.Background(), method, url, header, jbody, self.debug)
	_, respValue, err := httputils.ParseJSONResponse(bodystr, resp, err, self.debug)
	if err != nil {
		if e, ok := err.(*httputils.JSONClientError); ok && e.Code == 404 {
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
		}
		return nil, err
	}
	return respValue, err
}
