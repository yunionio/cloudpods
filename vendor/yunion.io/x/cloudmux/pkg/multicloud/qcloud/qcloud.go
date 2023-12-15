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

package qcloud

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/tencentyun/cos-go-sdk-v5/debug"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_QCLOUD    = api.CLOUD_PROVIDER_QCLOUD
	CLOUD_PROVIDER_QCLOUD_CN = "腾讯云"
	CLOUD_PROVIDER_QCLOUD_EN = "QCloud"

	QCLOUD_DEFAULT_REGION = "ap-beijing"

	QCLOUD_API_VERSION           = "2017-03-12"
	QCLOUD_CLB_API_VERSION       = "2018-03-17"
	QCLOUD_BILLING_API_VERSION   = "2018-07-09"
	QCLOUD_AUDIT_API_VERSION     = "2019-03-19"
	QCLOUD_CAM_API_VERSION       = "2019-01-16"
	QCLOUD_CDB_API_VERSION       = "2017-03-20"
	QCLOUD_MARIADB_API_VERSION   = "2017-03-12"
	QCLOUD_POSTGRES_API_VERSION  = "2017-03-12"
	QCLOUD_SQLSERVER_API_VERSION = "2018-03-28"
	QCLOUD_REDIS_API_VERSION     = "2018-04-12"
	QCLOUD_MEMCACHED_API_VERSION = "2019-03-18"
	QCLOUD_SSL_API_VERSION       = "2019-12-05"
	QCLOUD_CDN_API_VERSION       = "2018-06-06"
	QCLOUD_MONGODB_API_VERSION   = "2019-07-25"
	QCLOUD_ES_API_VERSION        = "2018-04-16"
	QCLOUD_DCDB_API_VERSION      = "2018-04-11"
	QCLOUD_KAFKA_API_VERSION     = "2019-08-19"
	QCLOUD_TKE_API_VERSION       = "2018-05-25"
	QCLOUD_DNS_API_VERSION       = "2021-03-23"
	QCLOUD_STS_API_VERSION       = "2018-08-13"
	QCLOUD_TAG_API_VERSION       = "2018-08-13"
)

type QcloudClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	secretId  string
	secretKey string
	appId     string

	debug bool
}

func NewQcloudClientConfig(secretId, secretKey string) *QcloudClientConfig {
	cfg := &QcloudClientConfig{
		secretId:  secretId,
		secretKey: secretKey,
	}
	return cfg
}

func (cfg *QcloudClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *QcloudClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *QcloudClientConfig) AppId(appId string) *QcloudClientConfig {
	cfg.appId = appId
	return cfg
}

func (cfg *QcloudClientConfig) Debug(debug bool) *QcloudClientConfig {
	cfg.debug = debug
	return cfg
}

type SQcloudClient struct {
	*QcloudClientConfig
	ownerId   string
	ownerName string

	iregions []cloudprovider.ICloudRegion
	ibuckets []cloudprovider.ICloudBucket
}

func NewQcloudClient(cfg *QcloudClientConfig) (*SQcloudClient, error) {
	client := SQcloudClient{
		QcloudClientConfig: cfg,
	}
	err := client.fetchRegions()
	if err != nil {
		return nil, errors.Wrap(err, "fetchRegions")
	}
	return &client, nil
}

// 默认接口请求频率限制：20次/秒
// 部分接口支持金融区地域。由于金融区和非金融区是隔离不互通的，因此当公共参数 Region 为金融区地域（例如 ap-shanghai-fsi）时，需要同时指定带金融区地域的域名，最好和 Region 的地域保持一致，例如：clb.ap-shanghai-fsi.tencentcloudapi.com
// https://cloud.tencent.com/document/product/416/6479
func apiDomain(product string, params map[string]string) string {
	regionId, _ := params["Region"]
	return apiDomainByRegion(product, regionId)
}

func apiDomainByRegion(product, regionId string) string {
	if strings.HasSuffix(regionId, "-fsi") {
		return product + "." + regionId + ".tencentcloudapi.com"
	} else {
		return product + ".tencentcloudapi.com"
	}
}

func jsonRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool, retry bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("cvm", params)
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params, updateFunc, debug, retry)
}

func tkeRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := "tke.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_TKE_API_VERSION, apiName, params, updateFunc, debug, true)
}

func vpcRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("vpc", params)
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params, updateFunc, debug, true)
}

func auditRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("cloudaudit", params)
	return _jsonRequest(client, domain, QCLOUD_AUDIT_API_VERSION, apiName, params, updateFunc, debug, true)
}

func cbsRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("cbs", params)
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params, updateFunc, debug, true)
}

// es
func esRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("es", params)
	return _jsonRequest(client, domain, QCLOUD_ES_API_VERSION, apiName, params, updateFunc, debug, true)
}

// kafka
func kafkaRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("ckafka", params)
	return _jsonRequest(client, domain, QCLOUD_KAFKA_API_VERSION, apiName, params, updateFunc, debug, true)
}

// redis
func redisRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("redis", params)
	return _jsonRequest(client, domain, QCLOUD_REDIS_API_VERSION, apiName, params, updateFunc, debug, true)
}

// tdsql
func dcdbRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("dcdb", params)
	return _jsonRequest(client, domain, QCLOUD_DCDB_API_VERSION, apiName, params, updateFunc, debug, true)
}

// mongodb
func mongodbRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("mongodb", params)
	return _jsonRequest(client, domain, QCLOUD_MONGODB_API_VERSION, apiName, params, updateFunc, debug, true)
}

// memcached
func memcachedRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("memcached", params)
	return _jsonRequest(client, domain, QCLOUD_MEMCACHED_API_VERSION, apiName, params, updateFunc, debug, true)
}

// loadbalancer服务 api 3.0
func clbRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("clb", params)
	return _jsonRequest(client, domain, QCLOUD_CLB_API_VERSION, apiName, params, updateFunc, debug, true)
}

// cdb
func cdbRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("cdb", params)
	return _jsonRequest(client, domain, QCLOUD_CDB_API_VERSION, apiName, params, updateFunc, debug, true)
}

// mariadb
func mariadbRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("mariadb", params)
	return _jsonRequest(client, domain, QCLOUD_MARIADB_API_VERSION, apiName, params, updateFunc, debug, true)
}

// postgres
func postgresRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("postgres", params)
	return _jsonRequest(client, domain, QCLOUD_POSTGRES_API_VERSION, apiName, params, updateFunc, debug, true)
}

// sqlserver
func sqlserverRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("sqlserver", params)
	return _jsonRequest(client, domain, QCLOUD_SQLSERVER_API_VERSION, apiName, params, updateFunc, debug, true)
}

// ssl 证书服务
func sslRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := "ssl.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_SSL_API_VERSION, apiName, params, updateFunc, debug, true)
}

// dnspod 解析服务
func dnsRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := "dnspod.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_DNS_API_VERSION, apiName, params, updateFunc, debug, true)
}

func billingRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := "billing.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_BILLING_API_VERSION, apiName, params, updateFunc, debug, true)
}

func camRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := "cam.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_CAM_API_VERSION, apiName, params, updateFunc, debug, true)
}

func monitorRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string),
	debug bool) (jsonutils.JSONObject, error) {
	domain := "monitor.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_API_VERSION_METRICS, apiName, params, updateFunc, debug, true)
}

func cdnRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string),
	debug bool) (jsonutils.JSONObject, error) {
	domain := "cdn.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_CDN_API_VERSION, apiName, params, updateFunc, debug, true)
}

func stsRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string),
	debug bool) (jsonutils.JSONObject, error) {
	domain := "sts.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_STS_API_VERSION, apiName, params, updateFunc, debug, true)
}

type qcloudResponse interface {
	tchttp.Response
	GetResponse() *interface{}
}

// 3.0版本通用response
type QcloudResponse struct {
	*tchttp.BaseResponse
	Response *interface{} `json:"Response"`
}

func (r *QcloudResponse) GetResponse() *interface{} {
	return r.Response
}

func _jsonRequest(client *common.Client, domain string, version string, apiName string, params map[string]string, updateFun func(string, string), debug bool, retry bool) (jsonutils.JSONObject, error) {
	req := &tchttp.BaseRequest{}
	_profile := profile.NewClientProfile()
	_profile.SignMethod = common.SHA256
	client.WithProfile(_profile)
	service := strings.Split(domain, ".")[0]
	req.Init().WithApiInfo(service, version, apiName)
	req.SetDomain(domain)

	for k, v := range params {
		if strings.HasSuffix(k, "Ids.0") && len(v) == 0 {
			return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s = %s", k, v)
		}
		req.GetParams()[k] = v
	}

	resp := &QcloudResponse{
		BaseResponse: &tchttp.BaseResponse{},
	}
	ret, err := _baseJsonRequest(client, req, resp, apiName, debug, retry)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNoPermission && updateFun != nil {
			updateFun(service, apiName)
		}
		return nil, err
	}
	return ret, nil
}

func _baseJsonRequest(client *common.Client, req tchttp.Request, resp qcloudResponse, apiName string, debug bool, retry bool) (jsonutils.JSONObject, error) {
	tryMax := 1
	if retry {
		tryMax = 3
	}
	var err error
	for i := 1; i <= tryMax; i++ {
		err = client.Send(req, resp)
		if err == nil {
			break
		}
		needRetry := false
		e, ok := err.(*sdkerrors.TencentCloudSDKError)
		if ok {
			if strings.HasPrefix(e.Code, "UnauthorizedOperation.") ||
				strings.HasPrefix(e.Code, "AuthFailure.") ||
				utils.IsInStringArray(e.Code, []string{
					"SecretidNotAuthAccessResource",
					"UnauthorizedOperation",
					"InvalidParameter.PermissionDenied",
					"AuthFailure",
				}) {
				return nil, errors.Wrapf(cloudprovider.ErrNoPermission, err.Error())
			}
			if utils.IsInStringArray(e.Code, []string{
				"AuthFailure.SecretIdNotFound",
				"AuthFailure.SignatureFailure",
			}) {
				return nil, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, err.Error())
			}
			if utils.IsInStringArray(e.Code, []string{
				"InvalidParameter.RoleNotExist",
				"ResourceNotFound",
				"FailedOperation.CertificateNotFound",
			}) {
				return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
			}

			if e.Code == "UnsupportedRegion" {
				return nil, cloudprovider.ErrNotSupported
			}
			if e.Code == "InvalidParameterValue" && apiName == "GetMonitorData" && strings.Contains(e.Message, "the instance has been destroyed") {
				return nil, cloudprovider.ErrNotFound
			}

			if utils.IsInStringArray(e.Code, []string{
				"InternalError",
				"MutexOperation.TaskRunning",       // Code=DesOperation.MutexTaskRunning, Message=Mutex task is running, please try later
				"InvalidInstance.NotSupported",     // Code=InvalidInstance.NotSupported, Message=The request does not support the instances `ins-bg54517v` which are in operation or in a special state., 重装系统后立即关机有可能会引发 Code=InvalidInstance.NotSupported 错误, 重试可以避免任务失败
				"InvalidAddressId.StatusNotPermit", // Code=InvalidAddressId.StatusNotPermit, Message=The status `"UNBINDING"` for AddressId `"eip-m3kix9kx"` is not permit to do this operation., EIP删除需要重试
				"RequestLimitExceeded",
				"OperationDenied.InstanceOperationInProgress", // 调整配置后开机 Code=OperationDenied.InstanceOperationInProgress, Message=实例`['ins-nksicizg']`操作进行中，请等待, RequestId=c9951005-b22c-43c1-84aa-d923d49addcf
			}) {
				needRetry = true
			}
		}

		if !needRetry {
			for _, msg := range []string{
				"EOF",
				"TLS handshake timeout",
				"try later",
				"i/o timeout",
			} {
				if strings.Contains(err.Error(), msg) {
					needRetry = true
					break
				}
			}
		}

		if needRetry {
			log.Errorf("request url %s\nparams: %s\nerror: %v\ntry after %d seconds", req.GetDomain(), jsonutils.Marshal(req.GetParams()).PrettyString(), err, i*10)
			time.Sleep(time.Second * time.Duration(i*10))
			continue
		}
		log.Errorf("request url: %s\nparams: %s\nresponse: %v\nerror: %v", req.GetDomain(), jsonutils.Marshal(req.GetParams()).PrettyString(), resp.GetResponse(), err)
		return nil, err
	}
	if debug {
		log.Debugf("request: %s", req.GetParams())
		response := resp.GetResponse()
		if response != nil {
			log.Debugf("response: %s", jsonutils.Marshal(response).PrettyString())
		}
	}
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(resp.GetResponse()), nil
}

func (client *SQcloudClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(client.iregions))
	for i := 0; i < len(regions); i++ {
		region := client.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (client *SQcloudClient) getDefaultClient(params map[string]string) (*common.Client, error) {
	regionId := QCLOUD_DEFAULT_REGION
	if len(params) > 0 {
		if region, ok := params["Region"]; ok {
			regionId = region
		}
	}
	return client.getSdkClient(regionId)
}

func (client *SQcloudClient) getSdkClient(regionId string) (*common.Client, error) {
	cli, err := common.NewClientWithSecretId(client.secretId, client.secretKey, regionId)
	if err != nil {
		return nil, err
	}
	httpClient := client.cpcfg.AdaptiveTimeoutHttpClient()
	ts, _ := httpClient.Transport.(*http.Transport)
	cli.WithHttpTransport(cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "ioutil.ReadAll")
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		params, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, errors.Wrapf(err, "ParseQuery(%s)", string(body))
		}
		service := strings.Split(req.URL.Host, ".")[0]
		action := params.Get("Action")
		respCheck := func(resp *http.Response) error {
			if client.cpcfg.UpdatePermission != nil {
				client.cpcfg.UpdatePermission(service, action)
			}
			return nil
		}
		if client.cpcfg.ReadOnly {
			for _, prefix := range []string{"Get", "List", "Describe"} {
				if strings.HasPrefix(action, prefix) {
					return respCheck, nil
				}
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, action)
		}
		return respCheck, nil
	}))
	return cli, nil
}

func (client *SQcloudClient) tkeRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return tkeRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) vpcRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return vpcRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) auditRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return auditRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) cbsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return cbsRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) tagRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return tagRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func tagRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := "tag.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_TAG_API_VERSION, apiName, params, updateFunc, debug, true)
}

func (client *SQcloudClient) clbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return clbRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) cdbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return cdbRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) esRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}

	return esRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) kafkaRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}

	return kafkaRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) redisRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}

	return redisRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) dcdbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}

	return dcdbRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) mongodbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}

	return mongodbRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) memcachedRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}

	return memcachedRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) mariadbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return mariadbRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) postgresRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return postgresRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) sqlserverRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return sqlserverRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) sslRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return sslRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) dnsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return dnsRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) billingRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return billingRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) camRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return camRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) cdnRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return cdnRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) stsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return stsRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

func (client *SQcloudClient) jsonRequest(apiName string, params map[string]string, retry bool) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug, retry)
}

func (client *SQcloudClient) fetchRegions() error {
	body, err := client.jsonRequest("DescribeRegions", nil, true)
	if err != nil {
		log.Errorf("fetchRegions fail %s", err)
		return err
	}

	regions := make([]SRegion, 0)
	err = body.Unmarshal(&regions, "RegionSet")
	if err != nil {
		log.Errorf("unmarshal json error %s", err)
		return err
	}
	client.iregions = make([]cloudprovider.ICloudRegion, len(regions))
	for i := 0; i < len(regions); i++ {
		regions[i].client = client
		client.iregions[i] = &regions[i]
	}
	return nil
}

func (client *SQcloudClient) getCosClient(bucket *SBucket) (*cos.Client, error) {
	var baseUrl *cos.BaseURL
	if bucket != nil {
		u, _ := url.Parse(bucket.getBucketUrl())
		baseUrl = &cos.BaseURL{
			BucketURL: u,
		}
	}
	ts := &http.Transport{
		Proxy: client.cpcfg.ProxyFunc,
	}
	cosClient := cos.NewClient(
		baseUrl,
		&http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  client.secretId,
				SecretKey: client.secretKey,
				Transport: &debug.DebugRequestTransport{
					RequestHeader:  client.debug,
					RequestBody:    client.debug,
					ResponseHeader: client.debug,
					ResponseBody:   client.debug,
					Transport: cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
						method, path := req.Method, req.URL.Path
						respCheck := func(resp *http.Response) error {
							if resp.StatusCode == 403 {
								if client.cpcfg.UpdatePermission != nil {
									client.cpcfg.UpdatePermission("cos", fmt.Sprintf("%s %s", method, path))
								}
							}
							return nil
						}
						if client.cpcfg.ReadOnly {
							if req.Method == "GET" || req.Method == "HEAD" {
								return respCheck, nil
							}
							return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
						}
						return respCheck, nil
					}),
				},
			},
		},
	)
	return cosClient, nil
}

func (self *SQcloudClient) invalidateIBuckets() {
	self.ibuckets = nil
}

func (self *SQcloudClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if self.ibuckets == nil {
		err := self.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return self.ibuckets, nil
}

func (client *SQcloudClient) verifyAppId() error {
	region, err := client.getDefaultRegion()
	if err != nil {
		return errors.Wrap(err, "getDefaultRegion")
	}
	bucket := SBucket{
		region: region,
		Name:   "yuniondocument",
	}
	cli, err := client.getCosClient(&bucket)
	if err != nil {
		return errors.Wrap(err, "getCosClient")
	}
	resp, err := cli.Bucket.Head(context.Background())
	if resp != nil {
		defer httputils.CloseResponse(resp.Response)
		if resp.StatusCode < 400 || resp.StatusCode == 404 {
			return nil
		}
		return errors.Error(fmt.Sprintf("invalid AppId: %d", resp.StatusCode))
	}
	return errors.Wrap(err, "Head")
}

func (client *SQcloudClient) fetchBuckets() error {
	coscli, err := client.getCosClient(nil)
	if err != nil {
		return errors.Wrap(err, "GetCosClient")
	}
	s, _, err := coscli.Service.Get(context.Background())
	if err != nil {
		return errors.Wrap(err, "coscli.Service.Get")
	}
	client.ownerId = s.Owner.ID
	client.ownerName = s.Owner.DisplayName

	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range s.Buckets {
		bInfo := s.Buckets[i]
		createAt, _ := timeutils.ParseTimeStr(bInfo.CreationDate)
		slashPos := strings.LastIndexByte(bInfo.Name, '-')
		appId := bInfo.Name[slashPos+1:]
		if appId != client.appId {
			log.Errorf("[%s %s] Inconsistent appId: %s expect %s", bInfo.Name, bInfo.Region, appId, client.appId)
		}
		name := bInfo.Name[:slashPos]
		region, err := client.getIRegionByRegionId(bInfo.Region)
		var zone cloudprovider.ICloudZone
		if err != nil {
			log.Errorf("fail to find region %s", bInfo.Region)
			// possibly a zone, try zone
			regionStr := func() string {
				info := strings.Split(bInfo.Region, "-")
				if num, _ := strconv.Atoi(info[len(info)-1]); num > 0 {
					return strings.TrimSuffix(bInfo.Region, fmt.Sprintf("-%d", num))
				}
				return bInfo.Region
			}()
			region, err = client.getIRegionByRegionId(regionStr)
			if err != nil {
				log.Errorf("fail to find region %s", regionStr)
				continue
			}
			zoneId := bInfo.Region
			zone, _ = region.(*SRegion).getZoneById(bInfo.Region)
			if zone != nil {
				zoneId = zone.GetId()
			}
			log.Debugf("find zonal bucket %s", zoneId)
		}
		b := SBucket{
			region:     region.(*SRegion),
			appId:      appId,
			Name:       name,
			Location:   bInfo.Region,
			CreateDate: createAt,
		}
		if zone != nil {
			b.zone = zone.(*SZone)
		}
		ret = append(ret, &b)
	}
	client.ibuckets = ret
	return nil
}

func (client *SQcloudClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	err := client.fetchRegions()
	if err != nil {
		return nil, err
	}
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = client.GetAccountId()
	subAccount.Name = client.cpcfg.Name
	subAccount.Account = client.secretId
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	subAccount.DefaultProjectId = "0"
	if len(client.appId) > 0 {
		subAccount.Account = fmt.Sprintf("%s/%s", client.secretId, client.appId)
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SQcloudClient) GetAccountId() string {
	if len(self.ownerId) == 0 {
		caller, err := self.GetCallerIdentity()
		if err == nil {
			self.ownerId = caller.AccountId
		}
	}
	return self.ownerId
}

func (client *SQcloudClient) GetIamLoginUrl() string {
	return fmt.Sprintf("https://cloud.tencent.com/login/subAccount")
}

func (client *SQcloudClient) GetIRegions() []cloudprovider.ICloudRegion {
	return client.iregions
}

func (client *SQcloudClient) getDefaultRegion() (*SRegion, error) {
	iregion, err := client.getIRegionByRegionId(QCLOUD_DEFAULT_REGION)
	if err != nil {
		return nil, errors.Wrap(err, "getIRegionByRegionId")
	}
	return iregion.(*SRegion), nil
}

func (client *SQcloudClient) getIRegionByRegionId(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(client.iregions); i++ {
		if client.iregions[i].GetId() == id {
			return client.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (client *SQcloudClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(client.iregions); i++ {
		if client.iregions[i].GetGlobalId() == id {
			return client.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (client *SQcloudClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(client.iregions); i++ {
		if client.iregions[i].GetId() == regionId {
			return client.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (client *SQcloudClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	for i := 0; i < len(client.iregions); i++ {
		ihost, err := client.iregions[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (client *SQcloudClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(client.iregions); i++ {
		ihost, err := client.iregions[i].GetIVpcById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (client *SQcloudClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(client.iregions); i++ {
		ihost, err := client.iregions[i].GetIStorageById(id)
		if err == nil {
			return ihost, nil
		} else if errors.Cause(err) != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

type SAccountBalance struct {
	Balance  float64
	Uin      int64
	Currency string
}

func (client *SQcloudClient) QueryAccountBalance() (*SAccountBalance, error) {
	body, err := client.billingRequest("DescribeAccountBalance", nil)
	if err != nil {
		if isError(err, []string{"UnauthorizedOperation.NotFinanceAuth"}) {
			return nil, cloudprovider.ErrNoBalancePermission
		}
		return nil, errors.Wrapf(err, "DescribeAccountBalance")
	}
	balance := &SAccountBalance{Currency: "CNY"}
	err = body.Unmarshal(balance)
	if err != nil {
		return nil, err
	}
	balance.Balance = balance.Balance / 100.0
	if balance.Uin >= 200000000000 {
		balance.Currency = "USD"
	}
	return balance, nil
}

func (client *SQcloudClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	projects := []SProject{}
	for {
		part, total, err := client.GetProjects(len(projects), 1000)
		if err != nil {
			return nil, errors.Wrap(err, "GetProjects")
		}
		projects = append(projects, part...)
		if len(projects) >= total || len(part) == 0 {
			break
		}
	}
	projects = append(projects, SProject{ProjectId: "0", ProjectName: "默认项目"})
	iprojects := []cloudprovider.ICloudProject{}
	for i := 0; i < len(projects); i++ {
		projects[i].client = client
		iprojects = append(iprojects, &projects[i])
	}
	return iprojects, nil
}

func (self *SQcloudClient) GetISSLCertificates() ([]cloudprovider.ICloudSSLCertificate, error) {
	rs, err := self.GetCertificates("", "", "")
	if err != nil {
		return nil, err
	}

	result := make([]cloudprovider.ICloudSSLCertificate, 0)
	for i := range rs {
		rs[i].client = self
		result = append(result, &rs[i])
	}
	return result, nil
}

func (self *SQcloudClient) GetCapabilities() []string {
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
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_DNSZONE,
		cloudprovider.CLOUD_CAPABILITY_PUBLIC_IP,
		cloudprovider.CLOUD_CAPABILITY_SAML_AUTH,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_MONGO_DB + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_ES + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_KAFKA + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CDN + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CONTAINER + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CERT,
		cloudprovider.CLOUD_CAPABILITY_SNAPSHOT_POLICY,
	}
	return caps
}
