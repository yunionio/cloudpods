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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	"yunion.io/x/pkg/util/timeutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_QCLOUD    = api.CLOUD_PROVIDER_QCLOUD
	CLOUD_PROVIDER_QCLOUD_CN = "腾讯云"

	QCLOUD_DEFAULT_REGION = "ap-beijing"

	QCLOUD_API_VERSION         = "2017-03-12"
	QCLOUD_CLB_API_VERSION     = "2018-03-17"
	QCLOUD_BILLING_API_VERSION = "2018-07-09"
	QCLOUD_AUDIT_API_VERSION   = "2019-03-19"
)

type SQcloudClient struct {
	providerId   string
	providerName string
	AppID        string
	SecretID     string
	SecretKey    string

	ownerId   string
	ownerName string

	iregions []cloudprovider.ICloudRegion
	ibuckets []cloudprovider.ICloudBucket

	Debug bool
}

func NewQcloudClient(providerId string, providerName string, secretID string, secretKey string, appID string, isDebug bool) (*SQcloudClient, error) {
	client := SQcloudClient{
		providerId:   providerId,
		providerName: providerName,
		SecretID:     secretID,
		SecretKey:    secretKey,
		AppID:        appID,
		Debug:        isDebug,
	}
	err := client.fetchRegions()
	if err != nil {
		return nil, errors.Wrap(err, "fetchRegions")
	}
	err = client.verifyAppId()
	if err != nil {
		return nil, errors.Wrap(err, "verifyAppId")
	}
	err = client.fetchBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "fetchBuckets")
	}
	if isDebug {
		log.Debugf("ownerID: %s ownerName: %s", client.ownerId, client.ownerName)
	}
	return &client, nil
}

// 默认接口请求频率限制：20次/秒
// 部分接口支持金融区地域。由于金融区和非金融区是隔离不互通的，因此当公共参数 Region 为金融区地域（例如 ap-shanghai-fsi）时，需要同时指定带金融区地域的域名，最好和 Region 的地域保持一致，例如：clb.ap-shanghai-fsi.tencentcloudapi.com
// https://cloud.tencent.com/document/product/416/6479
func apiDomain(product string, params map[string]string) string {
	region, ok := params["Region"]
	if ok && strings.HasSuffix(region, "-fsi") {
		return product + "." + region + ".tencentcloudapi.com"
	} else {
		return product + ".tencentcloudapi.com"
	}
}

func jsonRequest(client *common.Client, apiName string, params map[string]string, debug bool, retry bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("cvm", params)
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params, debug, retry)
}

func vpcRequest(client *common.Client, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("vpc", params)
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params, debug, true)
}

func auditRequest(client *common.Client, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("cloudaudit", params)
	return _jsonRequest(client, domain, QCLOUD_AUDIT_API_VERSION, apiName, params, debug, true)
}

func cbsRequest(client *common.Client, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("cbs", params)
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params, debug, true)
}

func accountRequest(client *common.Client, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	domain := "account.api.qcloud.com"
	return _phpJsonRequest(client, &wssJsonResponse{}, domain, "/v2/index.php", "", apiName, params, debug)
}

// loadbalancer服务 api 3.0
func clbRequest(client *common.Client, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	domain := apiDomain("clb", params)
	return _jsonRequest(client, domain, QCLOUD_CLB_API_VERSION, apiName, params, debug, true)
}

// loadbalancer服务 api 2017
func lbRequest(client *common.Client, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	domain := "lb.api.qcloud.com"
	return _phpJsonRequest(client, &lbJsonResponse{}, domain, "/v2/index.php", "", apiName, params, debug)
}

// ssl 证书服务
func wssRequest(client *common.Client, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	domain := "wss.api.qcloud.com"
	return _phpJsonRequest(client, &wssJsonResponse{}, domain, "/v2/index.php", "", apiName, params, debug)
}

// 2017版API
func vpc2017Request(client *common.Client, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	domain := "vpc.api.qcloud.com"
	return _phpJsonRequest(client, &vpc2017JsonResponse{}, domain, "/v2/index.php", "", apiName, params, debug)
}

func billingRequest(client *common.Client, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	domain := "billing.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_BILLING_API_VERSION, apiName, params, debug, true)
}

func monitorRequest(client *common.Client, apiName string, params map[string]string,
	debug bool) (jsonutils.JSONObject, error) {
	domain := "monitor.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_API_VERSION_METRICS, apiName, params, debug, true)
}

// ============phpJsonRequest============
type qcloudResponse interface {
	tchttp.Response
	GetResponse() *interface{}
}

type phpJsonRequest struct {
	tchttp.BaseRequest
	Path string
}

func (r *phpJsonRequest) GetUrl() string {
	url := r.BaseRequest.GetUrl()
	if url == "" {
		return url
	}

	index := strings.Index(url, "?")
	if index == -1 {
		// POST request
		return strings.TrimSuffix(url, "/") + r.Path
	}

	p1, p2 := url[:index], url[index:]
	p1 = strings.TrimSuffix(p1, "/")
	return p1 + r.Path + p2
}

func (r *phpJsonRequest) GetPath() string {
	return r.Path
}

// 2017vpc专用response
type vpc2017JsonResponse struct {
	Code       int          `json:"code"`
	CodeDesc   string       `json:"codeDesc"`
	Message    string       `json:"message"`
	TotalCount int          `json:"totalCount"`
	Response   *interface{} `json:"data"`
}

func (r *vpc2017JsonResponse) ParseErrorFromHTTPResponse(body []byte) (err error) {
	resp := &vpc2017JsonResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		return
	}
	if resp.Code != 0 {
		return sdkerrors.NewTencentCloudSDKError(resp.CodeDesc, resp.Message, "")
	}

	return nil
}

func (r *vpc2017JsonResponse) GetResponse() *interface{} {
	if r.Response == nil {
		result, _ := jsonutils.Parse([]byte(`{"data":[],"totalCount":0}`))
		return func(resp interface{}) *interface{} {
			return &resp
		}(result)
	}
	return func(resp interface{}) *interface{} {
		return &resp
	}(jsonutils.Marshal(r))
}

// SSL证书专用response
type wssJsonResponse struct {
	Code     int          `json:"code"`
	CodeDesc string       `json:"codeDesc"`
	Message  string       `json:"message"`
	Response *interface{} `json:"data"`
}

func (r *wssJsonResponse) ParseErrorFromHTTPResponse(body []byte) (err error) {
	resp := &wssJsonResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		return
	}
	if resp.Code != 0 {
		return sdkerrors.NewTencentCloudSDKError(resp.CodeDesc, resp.Message, "")
	}

	return nil
}

func (r *wssJsonResponse) GetResponse() *interface{} {
	return r.Response
}

// 2017版负载均衡API专用response
type lbJsonResponse struct {
	Response map[string]interface{}
}

func (r *lbJsonResponse) ParseErrorFromHTTPResponse(body []byte) (err error) {
	resp := &wssJsonResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		return
	}
	if resp.Code != 0 {
		return sdkerrors.NewTencentCloudSDKError(resp.CodeDesc, resp.Message, "")
	}

	// hook 由于目前只能从这个方法中拿到原始的body.这里将原始body hook 到 Response
	err = json.Unmarshal(body, &r.Response)
	if err != nil {
		return
	}

	return nil
}

func (r *lbJsonResponse) GetResponse() *interface{} {
	return func(resp interface{}) *interface{} {
		return &resp
	}(r.Response)
}

// 3.0版本通用response
type QcloudResponse struct {
	*tchttp.BaseResponse
	Response *interface{} `json:"Response"`
}

func (r *QcloudResponse) GetResponse() *interface{} {
	return r.Response
}

func _jsonRequest(client *common.Client, domain string, version string, apiName string, params map[string]string, debug bool, retry bool) (jsonutils.JSONObject, error) {
	req := &tchttp.BaseRequest{}
	if region, ok := params["Region"]; ok {
		client = client.Init(region)
	}
	client.WithProfile(profile.NewClientProfile())
	service := strings.Split(domain, ".")[0]
	req.Init().WithApiInfo(service, version, apiName)
	req.SetDomain(domain)

	for k, v := range params {
		if strings.HasSuffix(k, "Ids.0") && len(v) == 0 {
			return nil, cloudprovider.ErrNotFound
		}
		req.GetParams()[k] = v
	}

	resp := &QcloudResponse{
		BaseResponse: &tchttp.BaseResponse{},
	}
	return _baseJsonRequest(client, req, resp, debug, retry)
}

// 老版本腾讯云api。 适用于类似 https://cvm.api.qcloud.com/v2/index.php 这样的带/v2/index.php路径的接口
// todo: 添加自定义response参数
func _phpJsonRequest(client *common.Client, resp qcloudResponse, domain string, path string, version string, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	req := &phpJsonRequest{Path: path}
	if region, ok := params["Region"]; ok {
		client = client.Init(region)
	}
	client.WithProfile(profile.NewClientProfile())
	service := strings.Split(domain, ".")[0]
	req.Init().WithApiInfo(service, version, apiName)
	req.SetDomain(domain)

	for k, v := range params {
		if strings.HasSuffix(k, "Ids.0") && len(v) == 0 {
			return nil, cloudprovider.ErrNotFound
		}
		req.GetParams()[k] = v
	}

	return _baseJsonRequest(client, req, resp, debug, true)
}

func _baseJsonRequest(client *common.Client, req tchttp.Request, resp qcloudResponse, debug bool, retry bool) (jsonutils.JSONObject, error) {
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
		for _, msg := range []string{
			"EOF",
			"TLS handshake timeout",
			"Code=InternalError",
			"try later",
			"Code=MutexOperation.TaskRunning",   // Code=DesOperation.MutexTaskRunning, Message=Mutex task is running, please try later
			"Code=InvalidInstance.NotSupported", // Code=InvalidInstance.NotSupported, Message=The request does not support the instances `ins-bg54517v` which are in operation or in a special state., 重装系统后立即关机有可能会引发 Code=InvalidInstance.NotSupported 错误, 重试可以避免任务失败
			"i/o timeout",
			"Code=InvalidAddressId.StatusNotPermit", // Code=InvalidAddressId.StatusNotPermit, Message=The status `"UNBINDING"` for AddressId `"eip-m3kix9kx"` is not permit to do this operation., EIP删除需要重试
		} {
			if strings.Contains(err.Error(), msg) {
				needRetry = true
				break
			}
		}
		if strings.Contains(err.Error(), "Code=ResourceNotFound") {
			return nil, cloudprovider.ErrNotFound
		}
		if strings.Contains(err.Error(), "Code=UnsupportedRegion") {
			return nil, cloudprovider.ErrNotSupported
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
		log.Debugf("response: %s", jsonutils.Marshal(resp.GetResponse()).PrettyString())
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

func (client *SQcloudClient) getDefaultClient() (*common.Client, error) {
	return common.NewClientWithSecretId(client.SecretID, client.SecretKey, QCLOUD_DEFAULT_REGION)
}

func (client *SQcloudClient) vpcRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return vpcRequest(cli, apiName, params, client.Debug)
}

func (client *SQcloudClient) auditRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return auditRequest(cli, apiName, params, client.Debug)
}

func (client *SQcloudClient) cbsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return cbsRequest(cli, apiName, params, client.Debug)
}

func (client *SQcloudClient) accountRequestRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return accountRequest(cli, apiName, params, client.Debug)
}

func (client *SQcloudClient) clbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return clbRequest(cli, apiName, params, client.Debug)
}

func (client *SQcloudClient) lbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return lbRequest(cli, apiName, params, client.Debug)
}

func (client *SQcloudClient) wssRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return wssRequest(cli, apiName, params, client.Debug)
}

func (client *SQcloudClient) vpc2017Request(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return vpc2017Request(cli, apiName, params, client.Debug)
}

func (client *SQcloudClient) billingRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return billingRequest(cli, apiName, params, client.Debug)
}

func (client *SQcloudClient) jsonRequest(apiName string, params map[string]string, retry bool) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, apiName, params, client.Debug, retry)
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
	cosClient := cos.NewClient(
		baseUrl,
		&http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  client.SecretID,
				SecretKey: client.SecretKey,
				Transport: &debug.DebugRequestTransport{
					RequestHeader:  client.Debug,
					RequestBody:    client.Debug,
					ResponseHeader: client.Debug,
					ResponseBody:   client.Debug,
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
		name := bInfo.Name
		// name = name[:len(name)-len(result.Owner.ID)-1]
		name = name[:strings.LastIndexByte(name, '-')]
		region, err := client.getIRegionByRegionId(bInfo.Region)
		if err != nil {
			log.Errorf("fail to find region %s", bInfo.Region)
			continue
		}
		b := SBucket{
			region:     region.(*SRegion),
			Name:       name,
			Location:   bInfo.Region,
			CreateDate: createAt,
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
	subAccount.Name = client.providerName
	subAccount.Account = client.SecretID
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	if len(client.AppID) > 0 {
		subAccount.Account = fmt.Sprintf("%s/%s", client.SecretID, client.AppID)
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (client *SQcloudClient) GetAccountId() string {
	return client.ownerName
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
		} else if err != cloudprovider.ErrNotFound {
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
		} else if err != cloudprovider.ErrNotFound {
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
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

type SAccountBalance struct {
	AvailableAmount     float64
	AvailableCashAmount float64
	CreditAmount        float64
	MybankCreditAmount  float64
	Currency            string
}

func (client *SQcloudClient) QueryAccountBalance() (*SAccountBalance, error) {
	balance := SAccountBalance{}
	body, err := client.billingRequest("DescribeAccountBalance", nil)
	if err != nil {
		if isError(err, []string{"UnauthorizedOperation.NotFinanceAuth"}) {
			return nil, cloudprovider.ErrNoBalancePermission
		}
		log.Errorf("DescribeAccountBalance fail %s", err)
		return nil, err
	}
	log.Debugf("%s", body)
	balanceCent, _ := body.Float("Balance")
	balance.AvailableAmount = balanceCent / 100.0
	return &balance, nil
}

func (client *SQcloudClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	projects := []SProject{}
	params := map[string]string{"allList": "1"}
	body, err := client.accountRequestRequest("DescribeProject", params)
	if err != nil {
		return nil, err
	}
	if err := body.Unmarshal(&projects); err != nil {
		return nil, err
	}
	projects = append(projects, SProject{
		ProjectId:   "0",
		ProjectName: "默认项目",
		// CreateTime:  time.Time{},
	})
	iprojects := []cloudprovider.ICloudProject{}
	for i := 0; i < len(projects); i++ {
		projects[i].client = client
		iprojects = append(iprojects, &projects[i])
	}
	return iprojects, nil
}

func (self *SQcloudClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
	}
	return caps
}
