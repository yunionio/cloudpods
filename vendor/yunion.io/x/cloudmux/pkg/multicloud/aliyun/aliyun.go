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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	alierr "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	v "yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	ALIYUN_INTERNATIONAL_CLOUDENV = "InternationalCloud"
	ALIYUN_FINANCE_CLOUDENV       = "FinanceCloud"

	CLOUD_PROVIDER_ALIYUN    = api.CLOUD_PROVIDER_ALIYUN
	CLOUD_PROVIDER_ALIYUN_CN = "阿里云"
	CLOUD_PROVIDER_ALIYUN_EN = "Aliyun"

	ALIYUN_DEFAULT_REGION = "cn-hangzhou"

	ALIYUN_API_VERSION     = "2014-05-26"
	ALIYUN_API_VERSION_VPC = "2016-04-28"
	ALIYUN_API_VERSION_LB  = "2014-05-15"
	ALIYUN_API_VERSION_KVS = "2015-01-01"

	ALIYUN_API_VERSION_TRIAL = "2020-07-06"

	ALIYUN_BSS_API_VERSION = "2017-12-14"

	ALIYUN_RAM_API_VERSION      = "2015-05-01"
	ALIYUN_RDS_API_VERSION      = "2014-08-15"
	ALIYUN_RM_API_VERSION       = "2020-03-31"
	ALIYUN_STS_API_VERSION      = "2015-04-01"
	ALIYUN_PVTZ_API_VERSION     = "2018-01-01"
	ALIYUN_ALIDNS_API_VERSION   = "2015-01-09"
	ALIYUN_CBN_API_VERSION      = "2017-09-12"
	ALIYUN_CDN_API_VERSION      = "2018-05-10"
	ALIYUN_IMS_API_VERSION      = "2019-08-15"
	ALIYUN_NAS_API_VERSION      = "2017-06-26"
	ALIYUN_WAF_API_VERSION      = "2019-09-10"
	ALIYUN_MONGO_DB_API_VERSION = "2015-12-01"
	ALIYUN_ES_API_VERSION       = "2017-06-13"
	ALIYUN_KAFKA_API_VERSION    = "2019-09-16"
	ALIYUN_K8S_API_VERSION      = "2015-12-15"
	ALIYUN_OTS_API_VERSION      = "2016-06-20"

	ALIYUN_SERVICE_ECS      = "ecs"
	ALIYUN_SERVICE_VPC      = "vpc"
	ALIYUN_SERVICE_RDS      = "rds"
	ALIYUN_SERVICE_ES       = "es"
	ALIYUN_SERVICE_SLB      = "slb"
	ALIYUN_SERVICE_KVS      = "kvs"
	ALIYUN_SERVICE_NAS      = "nas"
	ALIYUN_SERVICE_CDN      = "cdn"
	ALIYUN_SERVICE_MONGO_DB = "mongodb"
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

	ownerId string

	nasEndpoints map[string]string
	vpcEndpoints map[string]string

	resourceGroups []SResourceGroup

	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket
}

func NewAliyunClient(cfg *AliyunClientConfig) (*SAliyunClient, error) {
	client := SAliyunClient{
		AliyunClientConfig: cfg,
		nasEndpoints:       map[string]string{},
		vpcEndpoints:       map[string]string{},
	}
	return &client, client.fetchRegions()
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
				"ErrorClusterNotFound",
			} {
				if strings.Contains(err.Error(), code) {
					return nil, errors.Wrap(cloudprovider.ErrNotFound, err.Error())
				}
			}
			if e, ok := errors.Cause(err).(*alierr.ServerError); ok {
				code := e.ErrorCode()
				switch code {
				case "InternalError":
					if apiName == "QueryAccountBalance" {
						return nil, errors.Wrapf(cloudprovider.ErrNoPermission, err.Error())
					}
					return nil, err
				case "InvalidAccessKeyId.NotFound",
					"InvalidAccessKeyId",
					"NoEnabledAccessKey",
					"InvalidAccessKeyId.Inactive",
					"Forbidden.AccessKeyDisabled",
					"Forbidden.AccessKey":
					return nil, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, err.Error())
				case "404 Not Found", "InstanceNotFound":
					return nil, errors.Wrap(cloudprovider.ErrNotFound, err.Error())
				case "OperationDenied.NoStock":
					return nil, errors.Wrapf(err, "所请求的套餐在指定的区域内已售罄;尝试其他套餐或选择其他区域和可用区。")
				case "InvalidInstance.NotSupported",
					"SignatureNonceUsed",                  // SignatureNonce 重复。每次请求的 SignatureNonce 在 15 分钟内不能重复。
					"BackendServer.configuring",           // 负载均衡的前一个配置项正在配置中，请稍后再试。
					"Operation.Conflict",                  // 您当前的操作可能与其他人的操作产生了冲突，请稍后重试。
					"OperationDenied.ResourceControl",     // 指定的区域处于资源控制中，请稍后再试。
					"ServiceIsStopping",                   // 监听正在停止，请稍后重试。
					"ProcessingSameRequest",               // 正在处理相同的请求。请稍后再试。
					"ResourceInOperating",                 // 当前资源正在操作中，请求稍后重试。
					"InvalidFileSystemStatus.Ordering",    // Message: The filesystem is ordering now, please check it later.
					"OperationUnsupported.EipNatBWPCheck": // create nat snat
					retry = true
				default:
					if strings.HasPrefix(code, "EntityNotExist.") || strings.HasSuffix(code, ".NotFound") || strings.HasSuffix(code, "NotExist") {
						if strings.HasPrefix(apiName, "Delete") {
							return jsonutils.NewDict(), nil
						}
						return nil, errors.Wrap(cloudprovider.ErrNotFound, err.Error())
					}
					return nil, err
				}
			} else {
				for _, code := range []string{
					"EOF",
					"i/o timeout",
					"TLS handshake timeout",
					"Client.Timeout exceeded while awaiting headers",
					"connection reset by peer",
					"server misbehaving",
					"try later",
					"Another operation is being performed", // Another operation is being performed on the DB instance or the DB instance is faulty(赋予RDS账号权限)
				} {
					if strings.Contains(err.Error(), code) {
						retry = true
						break
					}
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
	return resp, err
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
	method := requests.POST
	for prefix, _method := range map[string]string{
		"Get":            requests.GET,
		"Describe":       requests.GET,
		"List":           requests.GET,
		"Delete":         requests.DELETE,
		"UntagResources": requests.DELETE,
	} {
		if strings.HasPrefix(apiName, prefix) {
			method = _method
			break
		}
	}
	if strings.HasPrefix(domain, "elasticsearch") {
		req.Product = "elasticsearch"
		req.ServiceCode = "elasticsearch"
		pathPattern, ok := params["PathPattern"]
		if !ok {
			return nil, errors.Errorf("Roa request missing pathPattern")
		}
		delete(params, "PathPattern")
		req.PathPattern = pathPattern
		req.Method = method

		if method == requests.POST {
			//`{"ResourceIds":["123"],"ResourceType":"type","Tags":[{"key":"1","value":"2"}]}`
			body := map[string]interface{}{}
			resourceIds := []string{}
			tags := []map[string]string{}

			for k, v := range params {
				if strings.HasPrefix(k, "ResourceId") {
					resourceIds = append(resourceIds, v)
				} else if strings.HasPrefix(k, "Tag") && strings.Contains(k, "Key") {
					valueKey := fmt.Sprintf("Tag.%s.Value", k[4:len(k)-4])
					tags = append(tags, map[string]string{"key": v, "value": params[valueKey]})
				}
			}

			if len(resourceIds) > 0 {
				body["ResourceIds"] = resourceIds
			}
			if len(tags) > 0 {
				body["Tags"] = tags
			}
			body["ResourceType"] = params["ResourceType"]

			req.Content = []byte(jsonutils.Marshal(body).String())
			req.GetHeaders()["Content-Type"] = "application/json"
		} else {
			resourceIds := []string{}
			tagKeys := []string{}
			for k, v := range params {
				if strings.HasPrefix(k, "ResourceId") {
					resourceIds = append(resourceIds, v)
				} else if strings.HasPrefix(k, "TagKey") {
					tagKeys = append(tagKeys, v)
				}
			}
			if len(resourceIds) > 0 {
				req.QueryParams["ResourceIds"] = jsonutils.Marshal(resourceIds).String()
			}
			if len(tagKeys) > 0 {
				req.QueryParams["TagKeys"] = jsonutils.Marshal(tagKeys).String()
			}
		}

	} else if strings.HasPrefix(domain, "alikafka") { //alikafka DeleteInstance必须显式指定Method
		req.Method = requests.POST
	} else if strings.HasPrefix(domain, "cs") { //容器
		pathPattern, ok := params["PathPattern"]
		if !ok {
			return nil, errors.Errorf("Roa request missing pathPattern")
		}
		delete(params, "PathPattern")
		req.PathPattern = pathPattern
		req.Method = method
		req.GetHeaders()["Content-Type"] = "application/json"
	}

	resp, err := processCommonRequest(client, req)
	if err != nil {
		return nil, errors.Wrapf(err, "processCommonRequest with params %s", params)
	}
	body, err := jsonutils.Parse(resp.GetHttpContentBytes())
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Parse")
	}
	//{"Code":"InvalidInstanceType.ValueNotSupported","HostId":"ecs.aliyuncs.com","Message":"The specified instanceType beyond the permitted range.","RequestId":"0042EE30-0EDF-48A7-A414-56229D4AD532"}
	//{"Code":"200","Message":"successful","PageNumber":1,"PageSize":50,"RequestId":"BB4C970C-0E23-48DC-A3B0-EB21FFC70A29","RouterTableList":{"RouterTableListType":[{"CreationTime":"2017-03-19T13:37:40Z","Description":"","ResourceGroupId":"rg-acfmwie3cqoobmi","RouteTableId":"vtb-j6c60lectdi80rk5xz43g","RouteTableName":"","RouteTableType":"System","RouterId":"vrt-j6c00qrol733dg36iq4qj","RouterType":"VRouter","VSwitchIds":{"VSwitchId":["vsw-j6c3gig5ub4fmi2veyrus"]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}]},"Success":true,"TotalCount":1}
	//{"Code":"Success","Data":{"CashCoupon":[]},"Message":"Successful!","RequestId":"87AD7E9A-3F8F-460F-9934-FFFE502325EE","Success":true}
	if body.Contains("Code") {
		code, _ := body.GetString("Code")
		if len(code) > 0 && !utils.IsInStringArray(code, []string{"200", "Success"}) {
			return nil, fmt.Errorf(body.String())
		}
	}
	return body, nil
}

func (self *SAliyunClient) getNasEndpoint(regionId string) string {
	err := self.fetchNasEndpoints()
	if err != nil {
		return "nas.aliyuncs.com"
	}
	ep, ok := self.nasEndpoints[regionId]
	if ok && len(ep) > 0 {
		return ep
	}
	return "nas.aliyuncs.com"
}

func (self *SAliyunClient) fetchNasEndpoints() error {
	if len(self.nasEndpoints) > 0 {
		return nil
	}
	client, err := self.getDefaultClient()
	if err != nil {
		return errors.Wrapf(err, "getDefaultClient")
	}
	resp, err := jsonRequest(client, "nas.aliyuncs.com", ALIYUN_NAS_API_VERSION, "DescribeRegions", nil, self.debug)
	if err != nil {
		return errors.Wrapf(err, "DescribeRegions")
	}
	regions := []SRegion{}
	err = resp.Unmarshal(&regions, "Regions", "Region")
	if err != nil {
		return errors.Wrapf(err, "resp.Unmarshal")
	}
	for _, region := range regions {
		self.nasEndpoints[region.RegionId] = region.RegionEndpoint
	}
	return nil
}

func (self *SAliyunClient) getDefaultClient() (*sdk.Client, error) {
	client, err := self.getSdkClient(ALIYUN_DEFAULT_REGION)
	return client, err
}

func (self *SAliyunClient) getVpcEndpoint(regionId string) string {
	err := self.fetchVpcEndpoints()
	if err != nil {
		return "vpc.aliyuncs.com"
	}
	ep, ok := self.vpcEndpoints[regionId]
	if ok && len(ep) > 0 {
		return ep
	}
	return "vpc.aliyuncs.com"
}

func (self *SAliyunClient) fetchVpcEndpoints() error {
	if len(self.vpcEndpoints) > 0 {
		return nil
	}
	client, err := self.getDefaultClient()
	if err != nil {
		return errors.Wrapf(err, "getDefaultClient")
	}
	resp, err := jsonRequest(client, "vpc.aliyuncs.com", ALIYUN_API_VERSION_VPC, "DescribeRegions", nil, self.debug)
	if err != nil {
		return errors.Wrapf(err, "DescribeRegions")
	}
	regions := []SRegion{}
	err = resp.Unmarshal(&regions, "Regions", "Region")
	if err != nil {
		return errors.Wrapf(err, "resp.Unmarshal")
	}
	for _, region := range regions {
		self.vpcEndpoints[region.RegionId] = region.RegionEndpoint
	}
	return nil
}

func (self *SAliyunClient) getSdkClient(regionId string) (*sdk.Client, error) {
	transport := httputils.GetAdaptiveTransport(true)
	transport.Proxy = self.cpcfg.ProxyFunc
	client, err := sdk.NewClientWithOptions(
		regionId,
		&sdk.Config{
			HttpTransport: transport,
			Transport: cloudprovider.GetCheckTransport(transport, func(req *http.Request) (func(resp *http.Response), error) {
				params, err := url.ParseQuery(req.URL.RawQuery)
				if err != nil {
					return nil, errors.Wrapf(err, "ParseQuery(%s)", req.URL.RawQuery)
				}
				service := strings.Split(req.URL.Host, ".")[0]
				action := params.Get("Action")
				respCheck := func(resp *http.Response) {
					if self.cpcfg.UpdatePermission != nil && resp.StatusCode >= 400 && resp.ContentLength > 0 {
						body, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							return
						}
						resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
						obj, err := jsonutils.Parse(body)
						if err != nil {
							return
						}
						ret := struct{ Code string }{}
						obj.Unmarshal(&ret)
						if utils.IsInStringArray(ret.Code, []string{
							"NoPermission",
							"SubAccountNoPermission",
						}) || utils.HasPrefix(ret.Code, "Forbidden") ||
							action == "QueryAccountBalance" && ret.Code == "InternalError" {
							self.cpcfg.UpdatePermission(service, action)
						}
					}
				}
				for _, prefix := range []string{"Get", "List", "Describe", "Query"} {
					if strings.HasPrefix(action, prefix) {
						return respCheck, nil
					}
				}
				if self.cpcfg.ReadOnly {
					return respCheck, errors.Wrapf(cloudprovider.ErrAccountReadOnly, action)
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

func (self *SAliyunClient) imsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	params = self.SetResourceGropuId(params)
	return jsonRequest(cli, "ims.aliyuncs.com", ALIYUN_IMS_API_VERSION, apiName, params, self.debug)
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
	params = self.SetResourceGropuId(params)
	return jsonRequest(cli, "ecs.aliyuncs.com", ALIYUN_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) pvtzRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	params = self.SetResourceGropuId(params)
	return jsonRequest(cli, "pvtz.aliyuncs.com", ALIYUN_PVTZ_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) alidnsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	params = self.SetResourceGropuId(params)
	return jsonRequest(cli, "alidns.aliyuncs.com", ALIYUN_ALIDNS_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) cbnRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	params = self.SetResourceGropuId(params)
	return jsonRequest(cli, "cbn.aliyuncs.com", ALIYUN_CBN_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) cdnRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	params = self.SetResourceGropuId(params)
	return jsonRequest(cli, "cdn.aliyuncs.com", ALIYUN_CDN_API_VERSION, apiName, params, self.debug)
}

func (self *SAliyunClient) fetchRegions() error {
	body, err := self.ecsRequest("DescribeRegions", map[string]string{"AcceptLanguage": "zh-CN"})
	if err != nil {
		return errors.Wrapf(err, "DescribeRegions")
	}

	regions := make([]SRegion, 0)
	err = body.Unmarshal(&regions, "Regions", "Region")
	if err != nil {
		return errors.Wrapf(err, "resp.Unmarshal")
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
func (client *SAliyunClient) getOssClientByEndpoint(endpoint string) (*oss.Client, error) {
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
	httpClient.Transport = cloudprovider.GetCheckTransport(transport, func(req *http.Request) (func(resp *http.Response), error) {
		path, method := req.URL.Path, req.Method
		respCheck := func(resp *http.Response) {
			if client.cpcfg.UpdatePermission != nil && resp.StatusCode == 403 {
				client.cpcfg.UpdatePermission("oss", fmt.Sprintf("%s %s", method, path))
			}
		}
		if client.cpcfg.ReadOnly {
			if req.Method == "GET" || req.Method == "HEAD" {
				return respCheck, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.RawPath)
		}
		return respCheck, nil
	})
	cliOpts := []oss.ClientOption{
		oss.HTTPClient(httpClient),
	}
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}
	cli, err := oss.New(endpoint, client.accessKey, client.accessSecret, cliOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "oss.New")
	}
	return cli, nil
}

func (client *SAliyunClient) getOssClient(regionId string) (*oss.Client, error) {
	ep := getOSSExternalDomain(regionId)
	return client.getOssClientByEndpoint(ep)
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

	if len(self.ownerId) == 0 {
		self.ownerId = result.Owner.ID
	}

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
	projects, err := self.GetIProjects()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIProject")
	}
	for i := range projects {
		if projects[i].GetName() == "默认资源组" {
			subAccount.DefaultProjectId = projects[i].GetGlobalId()
			break
		}
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SAliyunClient) GetAccountId() string {
	if len(self.ownerId) > 0 {
		return self.ownerId
	}
	caller, err := self.GetCallerIdentity()
	if err != nil {
		return ""
	}
	self.ownerId = caller.AccountId
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

func (self *SAliyunClient) GetProjects() ([]SResourceGroup, error) {
	if len(self.resourceGroups) > 0 {
		return self.resourceGroups, nil
	}
	pageSize, pageNumber := 50, 1
	self.resourceGroups = []SResourceGroup{}
	for {
		parts, total, err := self.GetResourceGroups(pageNumber, pageSize)
		if err != nil {
			return nil, errors.Wrap(err, "GetResourceGroups")
		}
		self.resourceGroups = append(self.resourceGroups, parts...)
		if len(self.resourceGroups) >= total {
			break
		}
		pageNumber += 1
	}
	return self.resourceGroups, nil
}

func (self *SAliyunClient) SetResourceGropuId(params map[string]string) map[string]string {
	if params == nil {
		params = map[string]string{}
	}
	for _, groupId := range self.cpcfg.AliyunResourceGroupIds {
		if utils.IsInStringArray(groupId, self.GetResourceGroupIds()) {
			params["ResourceGroupId"] = groupId
		}
	}
	return params
}

func (self *SAliyunClient) GetResourceGroupIds() []string {
	ret := []string{}
	resourceGroups, err := self.GetProjects()
	if err != nil {
		return ret
	}
	for i := range resourceGroups {
		ret = append(ret, resourceGroups[i].Id)
	}
	return ret
}

func (self *SAliyunClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	resourceGroups, err := self.GetProjects()
	if err != nil {
		return nil, err
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
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_DNSZONE,
		cloudprovider.CLOUD_CAPABILITY_INTERVPCNETWORK,
		cloudprovider.CLOUD_CAPABILITY_SAML_AUTH,
		cloudprovider.CLOUD_CAPABILITY_NAT,
		cloudprovider.CLOUD_CAPABILITY_NAS,
		cloudprovider.CLOUD_CAPABILITY_WAF,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_MONGO_DB + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_ES + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_KAFKA + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CDN + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CONTAINER + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_TABLESTORE + cloudprovider.READ_ONLY_SUFFIX,
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
