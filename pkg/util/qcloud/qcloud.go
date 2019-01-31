package qcloud

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

const (
	CLOUD_PROVIDER_QCLOUD    = models.CLOUD_PROVIDER_QCLOUD
	CLOUD_PROVIDER_QCLOUD_CN = "腾讯云"

	QCLOUD_DEFAULT_REGION = "ap-beijing"

	QCLOUD_API_VERSION         = "2017-03-12"
	QCLOUD_CLB_API_VERSION     = "2018-03-17"
	QCLOUD_BILLING_API_VERSION = "2018-07-09"
)

type SQcloudClient struct {
	providerId   string
	providerName string
	AppID        string
	SecretID     string
	SecretKey    string
	iregions     []cloudprovider.ICloudRegion
}

func NewQcloudClient(providerId string, providerName string, secretID string, secretKey string) (*SQcloudClient, error) {
	client := SQcloudClient{providerId: providerId, providerName: providerName, SecretID: secretID, SecretKey: secretKey}
	if account := strings.Split(secretID, "/"); len(account) == 2 {
		client.SecretID = account[0]
		client.AppID = account[1]
	}
	err := client.fetchRegions()
	if err != nil {
		return nil, err
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

func jsonRequest(client *common.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	domain := apiDomain("cvm", params)
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params)
}

func vpcRequest(client *common.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	domain := apiDomain("vpc", params)
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params)
}

func cbsRequest(client *common.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	domain := apiDomain("cbs", params)
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params)
}

// loadbalancer服务
func clbRequest(client *common.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	domain := apiDomain("clb", params)
	return _jsonRequest(client, domain, QCLOUD_CLB_API_VERSION, apiName, params)
}

// ssl 证书服务
func wssRequest(client *common.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	domain := "wss.api.qcloud.com"
	return _wssJsonRequest(client, domain, "/v2/index.php", "", apiName, params)
}

func billingRequest(client *common.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	domain := "billing.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_BILLING_API_VERSION, apiName, params)
}

// ============wssJsonRequest============
type qcloudResponse interface {
	tchttp.Response
	GetResponse() *interface{}
}

type wssJsonRequest struct {
	tchttp.BaseRequest
	Path string
}

func (r *wssJsonRequest) GetUrl() string {
	url := r.BaseRequest.GetUrl()
	if url == "" {
		return url
	}

	index := strings.Index(url, "?")
	if index == -1 {
		// POST request
		url = strings.TrimSuffix(url, "/") + r.Path
	}

	p1, p2 := url[:index], url[index:]
	p1 = strings.TrimSuffix(p1, "/")
	return p1 + r.Path + p2
}

func (r *wssJsonRequest) GetPath() string {
	return r.Path
}

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
		return errors.NewTencentCloudSDKError(resp.CodeDesc, resp.Message, "")
	}

	return nil
}

func (r *wssJsonResponse) GetResponse() *interface{} {
	return r.Response
}

// ==================================

type QcloudResponse struct {
	*tchttp.BaseResponse
	Response *interface{} `json:"Response"`
}

func (r *QcloudResponse) GetResponse() *interface{} {
	return r.Response
}

func _jsonRequest(client *common.Client, domain string, version string, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
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

	return _baseJsonRequest(client, req, resp)
}

// wss 专用的
func _wssJsonRequest(client *common.Client, domain string, path string, version string, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	req := &wssJsonRequest{Path: path}
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

	resp := &wssJsonResponse{}
	return _baseJsonRequest(client, req, resp)
}

func _baseJsonRequest(client *common.Client, req tchttp.Request, resp qcloudResponse) (jsonutils.JSONObject, error) {
	for i := 1; i <= 3; i++ {
		err := client.Send(req, resp)
		if err == nil {
			break
		}
		needRetry := false
		for _, msg := range []string{"EOF", "TLS handshake timeout", "Code=InternalError", "retry later", "Code=MutexOperation.TaskRunning"} {
			if strings.Contains(err.Error(), msg) {
				needRetry = true
				break
			}
		}
		if needRetry && i != 3 {
			log.Errorf("request url %s\nparams: %s\nerror: %v\nafter %d second try again", req.GetDomain(), jsonutils.Marshal(req.GetParams()).PrettyString(), err, i*10)
			time.Sleep(time.Second * time.Duration(i*10))
			continue
		}
		log.Errorf("request url: %s\nparams: %s\nresponse: %s\nerror: %v", req.GetDomain(), jsonutils.Marshal(req.GetParams()).PrettyString(), resp.GetResponse(), err)
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
	return vpcRequest(cli, apiName, params)
}

func (client *SQcloudClient) cbsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return cbsRequest(cli, apiName, params)
}

func (client *SQcloudClient) clbRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return clbRequest(cli, apiName, params)
}

func (client *SQcloudClient) wssRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return wssRequest(cli, apiName, params)
}

func (client *SQcloudClient) billingRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return billingRequest(cli, apiName, params)
}

func (client *SQcloudClient) jsonRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, apiName, params)
}

func (client *SQcloudClient) fetchRegions() error {
	body, err := client.jsonRequest("DescribeRegions", nil)
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

func (client *SQcloudClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	err := client.fetchRegions()
	if err != nil {
		return nil, err
	}
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = client.providerName
	subAccount.Account = client.SecretID
	subAccount.HealthStatus = models.CLOUD_PROVIDER_HEALTH_NORMAL
	if len(client.AppID) > 0 {
		subAccount.Account = fmt.Sprintf("%s/%s", client.SecretID, client.AppID)
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (client *SQcloudClient) GetIRegions() []cloudprovider.ICloudRegion {
	return client.iregions
}

func (client *SQcloudClient) getDefaultRegion() (cloudprovider.ICloudRegion, error) {
	if len(client.iregions) > 0 {
		return client.iregions[0], nil
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
		log.Errorf("DescribeAccountBalance fail %s", err)
		return nil, err
	}
	log.Debugf("%s", body)
	balanceCent, _ := body.Float("Balance")
	balance.AvailableAmount = balanceCent / 100.0
	return &balance, nil
}
