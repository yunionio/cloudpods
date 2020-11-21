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
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	v "yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	ALIYUN_INTERNATIONAL_CLOUDENV = "InternationalCloud"
	ALIYUN_FINANCE_CLOUDENV       = "FinanceCloud"

	CLOUD_PROVIDER_ALIYUN    = api.CLOUD_PROVIDER_ALIYUN
	CLOUD_PROVIDER_ALIYUN_CN = "阿里云"

	ALIYUN_DEFAULT_REGION = "cn-hangzhou"

	ALIYUN_API_VERSION     = "2014-05-26"
	ALIYUN_API_VERSION_VPC = "2016-04-28"
	ALIYUN_API_VERSION_LB  = "2014-05-15"
	ALIYUN_API_VERSION_KVS = "2015-01-01"

	ALIYUN_API_VERSION_TRIAL = "2017-12-04"

	ALIYUN_BSS_API_VERSION = "2017-12-14"

	ALIYUN_RAM_API_VERSION    = "2015-05-01"
	ALIYUN_API_VERION_RDS     = "2014-08-15"
	ALIYUN_RM_API_VERSION     = "2020-03-31"
	ALIYUN_STS_API_VERSION    = "2015-04-01"
	ALIYUN_PVTZ_API_VERSION   = "2018-01-01"
	ALIYUN_ALIDNS_API_VERSION = "2015-01-09"
	ALIYUN_CBN_API_VERSION    = "2017-09-12"
)

var (
	// https://help.aliyun.com/document_detail/31837.html?spm=a2c4g.11186623.2.18.675f2b8cu8CN5K#concept-zt4-cvy-5db
	OSS_FINANCE_REGION_MAP = map[string]string{
		"cn-hzfinance":              "cn-hangzhou",
		"cn-shanghai-finance-1-pub": "cn-shanghai-finance-1",
		"cn-szfinance":              "cn-shenzhen-finance-1",

		"cn-hzjbp":              "cn-hangzhou",
		"cn-shanghai-finance-1": "cn-shanghai-finance-1",
		"cn-shenzhen-finance-1": "cn-shenzhen-finance-1",
	}
)

type AliyunClientConfig struct {
	cpcfg        cloudprovider.ProviderConfig
	cloudEnv     string // 服务区域 InternationalCloud | FinanceCloud
	accessKey    string
	accessSecret string
	debug        bool
}

func NewAliyunClientConfig(cloudEnv, accessKey, accessSecret string) *AliyunClientConfig {
	cfg := &AliyunClientConfig{
		cloudEnv:     cloudEnv,
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
	return cfg
}

func (cfg *AliyunClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *AliyunClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *AliyunClientConfig) Debug(debug bool) *AliyunClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg AliyunClientConfig) Copy() AliyunClientConfig {
	return cfg
}

type SAliyunClient struct {
	*AliyunClientConfig

	ownerId   string
	ownerName string

	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket
}

func NewAliyunClient(cfg *AliyunClientConfig) (*SAliyunClient, error) {
	client := SAliyunClient{
		AliyunClientConfig: cfg,
	}
	err := client.fetchRegions()
	if err != nil {
		return nil, errors.Wrap(err, "fetchRegions")
	}
	err = client.fetchBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "fetchBuckets")
	}
	if client.debug {
		log.Debugf("ClientID: %s ClientName: %s", client.ownerId, client.ownerName)
	}
	return &client, nil
}

func jsonRequest(client *sdk.Client, domain, apiVersion, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	if debug {
		log.Debugf("request %s %s %s %s", domain, apiVersion, apiName, params)
	}
	for i := 1; i < 4; i++ {
		resp, err := _jsonRequest(client, domain, apiVersion, apiName, params)
		retry := false
		if err != nil {
			for _, code := range []string{
				"InvalidAccessKeyId.NotFound",
			} {
				if strings.Contains(err.Error(), code) {
					return nil, err
				}
			}
			for _, code := range []string{"404 Not Found", "EntityNotExist.Role", "EntityNotExist.Group"} {
				if strings.Contains(err.Error(), code) {
					return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
				}
			}
			for _, code := range []string{
				"EOF",
				"i/o timeout",
				"TLS handshake timeout",
				"connection reset by peer",
				"server misbehaving",
				"SignatureNonceUsed",
				"InvalidInstance.NotSupported",
				"try later",
				"BackendServer.configuring",
				"Another operation is being performed", //Another operation is being performed on the DB instance or the DB instance is faulty(赋予RDS账号权限)
			} {
				if strings.Contains(err.Error(), code) {
					retry = true
					break
				}
			}
		}
		if retry {
			if debug {
				log.Debugf("Retry %d...", i)
			}
			time.Sleep(time.Second * time.Duration(i*10))
			continue
		}
		if debug {
			log.Debugf("Response: %s", resp)
		}
		return resp, err
	}
	return nil, fmt.Errorf("timeout for request %s params: %s", apiName, params)
}

func _jsonRequest(client *sdk.Client, domain string, version string, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	req := requests.NewCommonRequest()
	req.Domain = domain
	req.Version = version
	req.ApiName = apiName
	if params != nil {
		for k, v := range params {
			req.QueryParams[k] = v
		}
	}
	req.Scheme = "https"
	req.GetHeaders()["User-Agent"] = "vendor/yunion-OneCloud@" + v.Get().GitVersion

	resp, err := processCommonRequest(client, req)
	if err != nil {
		log.Errorf("request %s error %s with params %s", apiName, err, params)
		return nil, err
	}
	body, err := jsonutils.Parse(resp.GetHttpContentBytes())
	if err != nil {
		log.Errorf("parse json fail %s", err)
		return nil, err
	}
	//{"Code":"InvalidInstanceType.ValueNotSupported","HostId":"ecs.aliyuncs.com","Message":"The specified instanceType beyond the permitted range.","RequestId":"0042EE30-0EDF-48A7-A414-56229D4AD532"}
	//{"Code":"200","Message":"successful","PageNumber":1,"PageSize":50,"RequestId":"BB4C970C-0E23-48DC-A3B0-EB21FFC70A29","RouterTableList":{"RouterTableListType":[{"CreationTime":"2017-03-19T13:37:40Z","Description":"","ResourceGroupId":"rg-acfmwie3cqoobmi","RouteTableId":"vtb-j6c60lectdi80rk5xz43g","RouteTableName":"","RouteTableType":"System","RouterId":"vrt-j6c00qrol733dg36iq4qj","RouterType":"VRouter","VSwitchIds":{"VSwitchId":["vsw-j6c3gig5ub4fmi2veyrus"]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}]},"Success":true,"TotalCount":1}
	if body.Contains("Code") {
		code, _ := body.GetString("Code")
		if len(code) > 0 && !utils.IsInStringArray(code, []string{"200"}) {
			return nil, fmt.Errorf(body.String())
		}
	}
	return body, nil
}

func (self *SAliyunClient) getDefaultClient() (*sdk.Client, error) {
	client, err := self.getSdkClient(ALIYUN_DEFAULT_REGION)
	return client, err
}

func (self *SAliyunClient) getSdkClient(regionId string) (*sdk.Client, error) {
	transport := httputils.GetTransport(true)
	transport.Proxy = self.cpcfg.ProxyFunc
	client, err := sdk.NewClientWithOptions(
		regionId,
		&sdk.Config{
			HttpTransport: transport,
		},
		&credentials.BaseCredential{
			AccessKeyId:     self.accessKey,
			AccessKeySecret: self.accessSecret,
		},
	)
	return client, err
}

func (self *SAliyunClient) rmRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "resourcemanager.aliyuncs.com", ALIYUN_RM_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) ecsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "ecs.aliyuncs.com", ALIYUN_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) trialRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "actiontrail.cn-hangzhou.aliyuncs.com", ALIYUN_API_VERSION_TRIAL, apiName, params, self.debug)
}

func (self *SAliyunClient) pvtzRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "pvtz.aliyuncs.com", ALIYUN_PVTZ_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) alidnsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "alidns.aliyuncs.com", ALIYUN_ALIDNS_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) cbnRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "cbn.aliyuncs.com", ALIYUN_CBN_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) fetchRegions() error {
	body, err := self.ecsRequest("DescribeRegions", map[string]string{"AcceptLanguage": "zh-CN"})
	if err != nil {
		log.Errorf("fetchRegions fail %s", err)
		return err
	}

	regions := make([]SRegion, 0)
	err = body.Unmarshal(&regions, "Regions", "Region")
	if err != nil {
		log.Errorf("unmarshal json error %s", err)
		return err
	}
	self.iregions = make([]cloudprovider.ICloudRegion, len(regions))
	for i := 0; i < len(regions); i += 1 {
		regions[i].client = self
		self.iregions[i] = &regions[i]
	}
	return nil
}

// oss endpoint
// https://help.aliyun.com/document_detail/31837.html?spm=a2c4g.11186623.2.6.6E8ZkO
func getOSSExternalDomain(regionId string) string {
	return fmt.Sprintf("oss-%s.aliyuncs.com", regionId)
}

func getOSSInternalDomain(regionId string) string {
	return fmt.Sprintf("oss-%s-internal.aliyuncs.com", regionId)
}

// https://help.aliyun.com/document_detail/31837.html?spm=a2c4g.11186623.2.6.XqEgD1
func (client *SAliyunClient) getOssClient(regionId string) (*oss.Client, error) {
	// NOTE
	//
	// oss package as of version 20181116160301-c6838fdc33ed does not
	// respect http.ProxyFromEnvironment.
	//
	// The ClientOption Proxy, AuthProxy lacks the feature NO_PROXY has
	// which can be used to whitelist ips, domains from http_proxy,
	// https_proxy setting
	// oss use no timeout client so as to send/download large files
	httpClient := client.cpcfg.AdaptiveTimeoutHttpClient()
	cliOpts := []oss.ClientOption{
		oss.HTTPClient(httpClient),
	}
	ep := getOSSExternalDomain(regionId)
	cli, err := oss.New(ep, client.accessKey, client.accessSecret, cliOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "oss.New")
	}
	return cli, nil
}

func (self *SAliyunClient) getRegionByRegionId(id string) (cloudprovider.ICloudRegion, error) {
	_id, ok := OSS_FINANCE_REGION_MAP[id]
	if ok {
		id = _id
	}
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAliyunClient) invalidateIBuckets() {
	self.iBuckets = nil
}

func (self *SAliyunClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if self.iBuckets == nil {
		err := self.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return self.iBuckets, nil
}

func (self *SAliyunClient) fetchBuckets() error {
	osscli, err := self.getOssClient(ALIYUN_DEFAULT_REGION)
	if err != nil {
		return errors.Wrap(err, "self.getOssClient")
	}
	result, err := osscli.ListBuckets()
	if err != nil {
		return errors.Wrap(err, "oss.ListBuckets")
	}

	self.ownerId = result.Owner.ID
	self.ownerName = result.Owner.DisplayName

	ret := make([]cloudprovider.ICloudBucket, 0)
	for _, bInfo := range result.Buckets {
		regionId := bInfo.Location[4:]
		region, err := self.getRegionByRegionId(regionId)
		if err != nil {
			log.Errorf("cannot find bucket's region %s", regionId)
			continue
		}
		b := SBucket{
			region:       region.(*SRegion),
			Name:         bInfo.Name,
			Location:     bInfo.Location,
			CreationDate: bInfo.CreationDate,
			StorageClass: bInfo.StorageClass,
		}
		ret = append(ret, &b)
	}
	self.iBuckets = ret
	return nil
}

func (self *SAliyunClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SAliyunClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	err := self.fetchRegions()
	if err != nil {
		return nil, err
	}
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SAliyunClient) GetAccountId() string {
	return self.ownerId
}

func (self *SAliyunClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SAliyunClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAliyunClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = ALIYUN_DEFAULT_REGION
	}
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SAliyunClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
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

func (self *SAliyunClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIVpcById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAliyunClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIStorageById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAliyunClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	pageSize, pageNumber := 50, 1
	resourceGroups := []SResourceGroup{}
	for {
		parts, total, err := self.GetResourceGroups(pageNumber, pageSize)
		if err != nil {
			return nil, errors.Wrap(err, "GetResourceGroups")
		}
		resourceGroups = append(resourceGroups, parts...)
		if len(resourceGroups) >= total {
			break
		}
		pageNumber += 1
	}
	ret := []cloudprovider.ICloudProject{}
	for i := range resourceGroups {
		ret = append(ret, &resourceGroups[i])
	}
	return ret, nil
}

func (region *SAliyunClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_DNSZONE,
		cloudprovider.CLOUD_CAPABILITY_INTERVPCNETWORK,
	}
	return caps
}

func (self *SAliyunClient) GetAccessEnv() string {
	switch self.cloudEnv {
	case ALIYUN_INTERNATIONAL_CLOUDENV:
		return api.CLOUD_ACCESS_ENV_ALIYUN_GLOBAL
	case ALIYUN_FINANCE_CLOUDENV:
		return api.CLOUD_ACCESS_ENV_ALIYUN_FINANCE
	default:
		return api.CLOUD_ACCESS_ENV_ALIYUN_GLOBAL
	}
}
