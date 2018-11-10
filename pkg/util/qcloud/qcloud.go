package qcloud

import (
	"fmt"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
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

	QCLOUD_API_VERSION = "2017-03-12"
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

func jsonRequest(client *common.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	domain := "cvm.tencentcloudapi.com"
	if region, ok := params["Region"]; ok && strings.HasSuffix(region, "-fsi") {
		domain = "cvm." + region + ".tencentcloudapi.com"
	}
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params)
}

func vpcRequest(client *common.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	domain := "vpc.tencentcloudapi.com"
	// if region, ok := params["Region"]; ok && strings.HasSuffix(region, "-fsi") {
	// 	domain = "vpc." + region + ".tencentcloudapi.com"
	// }
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params)
}

func cbsRequest(client *common.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	domain := "cbs.tencentcloudapi.com"
	if region, ok := params["Region"]; ok && strings.HasSuffix(region, "-fsi") {
		domain = "cbs." + region + ".tencentcloudapi.com"
	}
	return _jsonRequest(client, domain, QCLOUD_API_VERSION, apiName, params)
}

type QcloudResponse struct {
	*tchttp.BaseResponse
	Response *interface{} `json:"Response"`
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
		req.GetParams()[k] = v
	}
	resp := &QcloudResponse{
		BaseResponse: &tchttp.BaseResponse{},
	}
	err := client.Send(req, resp)
	if err != nil {
		log.Errorf("request url: %s\nparams: %s\nerror: %v", req.GetDomain(), jsonutils.Marshal(req.GetParams()).PrettyString(), err)
		return nil, err
	}
	return jsonutils.Marshal(resp.Response), nil
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

func (client *SQcloudClient) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	for i := 0; i < len(client.iregions); i++ {
		ihost, err := client.iregions[i].GetIStoragecacheById(id)
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
	return nil, nil
}
