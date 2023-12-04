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

package apsara

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"
	v "yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_APSARA    = api.CLOUD_PROVIDER_APSARA
	CLOUD_PROVIDER_APSARA_CN = "阿里云专有云"
	CLOUD_PROVIDER_APSARA_EN = "Aliyun Apsara"

	APSARA_API_VERSION     = "2014-05-26"
	APSARA_API_VERSION_VPC = "2016-04-28"
	APSARA_API_VERSION_LB  = "2014-05-15"
	APSARA_API_VERSION_KVS = "2015-01-01"

	APSARA_API_VERSION_TRIAL = "2017-12-04"

	APSARA_BSS_API_VERSION = "2017-12-14"

	APSARA_RAM_API_VERSION  = "2015-05-01"
	APSARA_API_VERION_RDS   = "2014-08-15"
	APSARA_ASCM_API_VERSION = "2019-05-10"
	APSARA_STS_API_VERSION  = "2015-04-01"
	APSARA_OTS_API_VERSION  = "2016-06-20"

	APSARA_PRODUCT_METRICS      = "Cms"
	APSARA_PRODUCT_RDS          = "Rds"
	APSARA_PRODUCT_VPC          = "Vpc"
	APSARA_PRODUCT_KVSTORE      = "R-kvstore"
	APSARA_PRODUCT_SLB          = "Slb"
	APSARA_PRODUCT_ECS          = "Ecs"
	APSARA_PRODUCT_ACTION_TRIAL = "actiontrail"
	APSARA_PRODUCT_STS          = "Sts"
	APSARA_PRODUCT_RAM          = "Ram"
	APSARA_PRODUCT_ASCM         = "ascm"
	APSARA_PRODUCT_OTS          = "ots"
)

type ApsaraClientConfig struct {
	cpcfg          cloudprovider.ProviderConfig
	accessKey      string
	accessSecret   string
	organizationId string
	debug          bool
}

func NewApsaraClientConfig(accessKey, accessSecret string, endpoint string) *ApsaraClientConfig {
	cfg := &ApsaraClientConfig{
		accessKey:      accessKey,
		accessSecret:   accessSecret,
		organizationId: "1",
	}
	if info := strings.Split(accessKey, "/"); len(info) == 2 {
		cfg.accessKey, cfg.organizationId = info[0], info[1]
	}
	return cfg
}

func (cfg *ApsaraClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *ApsaraClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *ApsaraClientConfig) Debug(debug bool) *ApsaraClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg ApsaraClientConfig) Copy() ApsaraClientConfig {
	return cfg
}

type SApsaraClient struct {
	*ApsaraClientConfig

	ownerId   string
	ownerName string

	departments []string

	iregions []cloudprovider.ICloudRegion
}

func NewApsaraClient(cfg *ApsaraClientConfig) (*SApsaraClient, error) {
	client := SApsaraClient{
		ApsaraClientConfig: cfg,
	}

	err := client.fetchRegions()
	if err != nil {
		return nil, errors.Wrap(err, "fetchRegions")
	}
	return &client, nil
}

func (self *SApsaraClient) getDomain(product string) string {
	return self.cpcfg.URL
}

func productRequest(client *sdk.Client, product, domain, apiVersion, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	params["Product"] = product
	return jsonRequest(client, domain, apiVersion, apiName, params, debug)
}

func jsonRequest(client *sdk.Client, domain, apiVersion, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	if debug {
		log.Debugf("request %s %s %s %s", domain, apiVersion, apiName, params)
	}
	var resp jsonutils.JSONObject
	var err error
	for i := 1; i < 4; i++ {
		resp, err = _jsonRequest(client, domain, apiVersion, apiName, params)
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
	return resp, errors.Wrapf(err, "jsonRequest")
}

func _jsonRequest(client *sdk.Client, domain string, version string, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	req := requests.NewCommonRequest()
	req.Domain = domain
	req.Version = version
	req.ApiName = apiName
	req.Scheme = "http"
	req.Method = "POST"
	if strings.HasPrefix(domain, "public.") {
		req.Scheme = "https"
	}
	id := ""
	if params != nil {
		for k, v := range params {
			if strings.HasPrefix(k, "x-acs-") {
				req.GetHeaders()[k] = v
				continue
			}
			req.QueryParams[k] = v
			if strings.ToLower(k) != "regionid" && strings.HasSuffix(k, "Id") {
				id = v
			}
		}
	}
	req.GetHeaders()["User-Agent"] = "vendor/yunion-OneCloud@" + v.Get().GitVersion
	if strings.HasPrefix(apiName, "Describe") && len(id) > 0 {
		req.GetHeaders()["x-acs-instanceId"] = id
	}

	resp, err := processCommonRequest(client, req)
	if err != nil {
		return nil, errors.Wrapf(err, "processCommonRequest(%s, %s)", apiName, params)
	}
	body, err := jsonutils.Parse(resp.GetHttpContentBytes())
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Parse")
	}
	//{"Code":"InvalidInstanceType.ValueNotSupported","HostId":"ecs.apsaracs.com","Message":"The specified instanceType beyond the permitted range.","RequestId":"0042EE30-0EDF-48A7-A414-56229D4AD532"}
	//{"Code":"200","Message":"successful","PageNumber":1,"PageSize":50,"RequestId":"BB4C970C-0E23-48DC-A3B0-EB21FFC70A29","RouterTableList":{"RouterTableListType":[{"CreationTime":"2017-03-19T13:37:40Z","Description":"","ResourceGroupId":"rg-acfmwie3cqoobmi","RouteTableId":"vtb-j6c60lectdi80rk5xz43g","RouteTableName":"","RouteTableType":"System","RouterId":"vrt-j6c00qrol733dg36iq4qj","RouterType":"VRouter","VSwitchIds":{"VSwitchId":["vsw-j6c3gig5ub4fmi2veyrus"]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}]},"Success":true,"TotalCount":1}
	if body.Contains("Code") {
		code, _ := body.GetString("Code")
		if len(code) > 0 && !utils.IsInStringArray(code, []string{"200"}) {
			return nil, fmt.Errorf(body.String())
		}
	}
	if body.Contains("errorKey") {
		return nil, errors.Errorf(body.String())
	}
	return body, nil
}

func (self *SApsaraClient) getDefaultClient(regionId string) (*sdk.Client, error) {
	if len(self.iregions) > 0 && len(regionId) == 0 {
		regionId = self.iregions[0].GetId()
	}
	transport := httputils.GetTransport(true)
	transport.Proxy = self.cpcfg.ProxyFunc
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client, err := sdk.NewClientWithOptions(
		regionId,
		&sdk.Config{
			HttpTransport: transport,
			Transport: cloudprovider.GetCheckTransport(transport, func(req *http.Request) (func(resp *http.Response) error, error) {
				params, err := url.ParseQuery(req.URL.RawQuery)
				if err != nil {
					return nil, errors.Wrapf(err, "ParseQuery(%s)", req.URL.RawQuery)
				}
				action := params.Get("OpenApiAction")
				if len(action) == 0 {
					action = params.Get("Action")
				}
				service := strings.ToLower(params.Get("Product"))
				respCheck := func(resp *http.Response) error {
					if self.cpcfg.UpdatePermission != nil {
						body, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							return nil
						}
						resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
						obj, err := jsonutils.Parse(body)
						if err != nil {
							return nil
						}
						ret := struct {
							AsapiErrorCode string `json:"asapiErrorCode"`
							Code           string
						}{}
						obj.Unmarshal(&ret)
						if ret.Code == "403" ||
							strings.Contains(ret.AsapiErrorCode, "NoPermission") ||
							utils.HasPrefix(ret.Code, "Forbidden") ||
							utils.HasPrefix(ret.Code, "NoPermission") {
							self.cpcfg.UpdatePermission(service, action)
						}
					}
					return nil
				}
				if self.cpcfg.ReadOnly && len(action) > 0 {
					for _, prefix := range []string{"Get", "List", "Describe"} {
						if strings.HasPrefix(action, prefix) {
							return respCheck, nil
						}
					}
					return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, action)
				}
				return respCheck, nil
			}),
		},
		&credentials.BaseCredential{
			AccessKeyId:     self.accessKey,
			AccessKeySecret: self.accessSecret,
		},
	)
	return client, err
}

func (self *SApsaraClient) ascmRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient("")
	if err != nil {
		return nil, err
	}
	return productRequest(cli, APSARA_PRODUCT_ASCM, self.cpcfg.URL, APSARA_ASCM_API_VERSION, apiName, params, self.debug)
}

func (self *SApsaraClient) ecsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient("")
	if err != nil {
		return nil, err
	}
	domain := self.getDomain(APSARA_PRODUCT_ECS)
	return productRequest(cli, APSARA_PRODUCT_ECS, domain, APSARA_API_VERSION, apiName, params, self.debug)
}

func (self *SApsaraClient) getAccountInfo() string {
	account := map[string]string{
		"aliyunPk":         "",
		"accountStructure": "",
		"parentPk":         "26842",
		"accessKeyId":      self.accessKey,
		"accessKeySecret":  self.accessSecret,
		"partnerPk":        "",
		"sourceIp":         "",
		"securityToken":    "",
	}
	return base64.StdEncoding.EncodeToString([]byte(jsonutils.Marshal(account).String()))
}

func (self *SApsaraClient) ossRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient("")
	if err != nil {
		return nil, err
	}
	//pm := map[string]string{}
	//for k, v := range params {
	//	if k != "RegionId" {
	//		pm[k] = v
	//		delete(params, k)
	//	}
	//}
	//if len(pm) > 0 {
	//	params["Params"] = jsonutils.Marshal(pm).String()
	//}
	if _, ok := params["RegionId"]; !ok {
		params["RegionId"] = self.cpcfg.DefaultRegion
	}
	params["ProductName"] = "oss"
	params["OpenApiAction"] = apiName
	return productRequest(cli, "OneRouter", self.cpcfg.URL, "2018-12-12", "DoOpenApi", params, self.debug)
}

func (self *SApsaraClient) trialRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient("")
	if err != nil {
		return nil, err
	}
	domain := self.getDomain(APSARA_PRODUCT_ACTION_TRIAL)
	return productRequest(cli, APSARA_PRODUCT_ACTION_TRIAL, domain, APSARA_API_VERSION_TRIAL, apiName, params, self.debug)
}

func (self *SApsaraClient) fetchRegions() error {
	params := map[string]string{"AcceptLanguage": "zh-CN"}
	if len(self.cpcfg.DefaultRegion) > 0 {
		params["RegionId"] = self.cpcfg.DefaultRegion
	}
	body, err := self.ecsRequest("DescribeRegions", params)
	if err != nil {
		return errors.Wrapf(err, "DescribeRegions")
	}

	regions := make([]SRegion, 0)
	err = body.Unmarshal(&regions, "Regions", "Region")
	if err != nil {
		return errors.Wrapf(err, "body.Unmarshal")
	}
	self.iregions = make([]cloudprovider.ICloudRegion, len(regions))
	for i := 0; i < len(regions); i += 1 {
		regions[i].client = self
		self.iregions[i] = &regions[i]
	}
	return nil
}

// https://help.apsara.com/document_detail/31837.html?spm=a2c4g.11186623.2.6.XqEgD1
func (client *SApsaraClient) getOssClient(endpoint string) (*oss.Client, error) {
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
	transport, _ := httpClient.Transport.(*http.Transport)
	httpClient.Transport = cloudprovider.GetCheckTransport(transport, func(req *http.Request) (func(resp *http.Response) error, error) {
		if client.cpcfg.ReadOnly {
			if req.Method == "GET" || req.Method == "HEAD" {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	cliOpts := []oss.ClientOption{
		oss.HTTPClient(httpClient),
	}
	cli, err := oss.New(endpoint, client.accessKey, client.accessSecret, cliOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "oss.New")
	}
	return cli, nil
}

func (self *SApsaraClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SApsaraClient) GetProvider() string {
	return self.cpcfg.Vendor
}

func (self *SApsaraClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	err := self.fetchRegions()
	if err != nil {
		return nil, err
	}
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = self.GetAccountId()
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKey
	if self.organizationId != "1" {
		subAccount.Account = fmt.Sprintf("%s/%s", self.accessKey, self.organizationId)
	}
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SApsaraClient) GetAccountId() string {
	return self.cpcfg.URL
}

func (self *SApsaraClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SApsaraClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SApsaraClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SApsaraClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
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

func (self *SApsaraClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
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

func (self *SApsaraClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
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

func (region *SApsaraClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_IPV6_GATEWAY + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_TABLESTORE + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}
