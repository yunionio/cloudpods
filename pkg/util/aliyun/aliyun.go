package aliyun

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"

	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

const (
	CLOUD_PROVIDER_ALIYUN    = models.CLOUD_PROVIDER_ALIYUN
	CLOUD_PROVIDER_ALIYUN_CN = "阿里云"

	ALIYUN_DEFAULT_REGION = "cn-hangzhou"

	ALIYUN_API_VERSION = "2014-05-26"

	ALIYUN_BSS_API_VERSION = "2017-12-14"
)

type SAliyunClient struct {
	providerId   string
	providerName string
	accessKey    string
	secret       string
	iregions     []cloudprovider.ICloudRegion
}

func NewAliyunClient(providerId string, providerName string, accessKey string, secret string) (*SAliyunClient, error) {
	client := SAliyunClient{providerId: providerId, providerName: providerName, accessKey: accessKey, secret: secret}
	err := client.fetchRegions()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func jsonRequest(client *sdk.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return _jsonRequest(client, "ecs.aliyuncs.com", ALIYUN_API_VERSION, apiName, params)
}

func businessRequest(client *sdk.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	return _jsonRequest(client, "business.aliyuncs.com", ALIYUN_BSS_API_VERSION, apiName, params)
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

	resp, err := client.ProcessCommonRequest(req)
	if err != nil {
		log.Errorf("request error %s", err)
		return nil, err
	}
	body, err := jsonutils.Parse(resp.GetHttpContentBytes())
	if err != nil {
		log.Errorf("parse json fail %s", err)
		return nil, err
	}
	return body, nil
}

func (self *SAliyunClient) UpdateAccount(accessKey, secret string) error {
	if self.accessKey != accessKey || self.secret != secret {
		self.accessKey = accessKey
		self.secret = secret
		return self.fetchRegions()
	} else {
		return nil
	}
}

func (self *SAliyunClient) getDefaultClient() (*sdk.Client, error) {
	return sdk.NewClientWithAccessKey(ALIYUN_DEFAULT_REGION, self.accessKey, self.secret)
}

func (self *SAliyunClient) jsonRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, apiName, params)
}

func (self *SAliyunClient) businessRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return businessRequest(cli, apiName, params)
}

func (self *SAliyunClient) fetchRegions() error {
	body, err := self.jsonRequest("DescribeRegions", nil)
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

func (self *SAliyunClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
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
		} else if err != cloudprovider.ErrNotFound {
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
		} else if err != cloudprovider.ErrNotFound {
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
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAliyunClient) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIStoragecacheById(id)
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

type SCashCoupon struct {
	ApplicableProducts  string
	ApplicableScenarios string
	Balance             float64
	CashCouponId        string
	CashCouponNo        string
	EffectiveTime       time.Time
	ExpiryTime          time.Time
	GrantedTime         time.Time
	NominalValue        float64
	Status              string
}

type SPrepaidCard struct {
	PrepaidCardId       string
	PrepaidCardNo       string
	GrantedTime         time.Time
	EffectiveTime       time.Time
	ExpiryTime          time.Time
	NominalValue        float64
	Balance             float64
	ApplicableProducts  string
	ApplicableScenarios string
}

func (self *SAliyunClient) QueryAccountBalance() (*SAccountBalance, error) {
	body, err := self.businessRequest("QueryAccountBalance", nil)
	if err != nil {
		log.Errorf("QueryAccountBalance fail %s", err)
		return nil, err
	}
	balance := SAccountBalance{}
	err = body.Unmarshal(&balance, "Data")
	if err != nil {
		log.Errorf("Unmarshal AccountBalance fail %s", err)
		return nil, err
	}
	return &balance, nil
}

func (self *SAliyunClient) QueryCashCoupons() ([]SCashCoupon, error) {
	params := make(map[string]string)
	params["EffectiveOrNot"] = "True"
	body, err := self.businessRequest("QueryCashCoupons", params)
	if err != nil {
		log.Errorf("QueryCashCoupons fail %s", err)
		return nil, err
	}
	coupons := make([]SCashCoupon, 0)
	err = body.Unmarshal(&coupons, "Data", "CashCoupon")
	if err != nil {
		log.Errorf("Unmarshal fail %s", err)
		return nil, err
	}
	return coupons, nil
}

func (self *SAliyunClient) QueryPrepaidCards() ([]SPrepaidCard, error) {
	params := make(map[string]string)
	params["EffectiveOrNot"] = "True"
	body, err := self.businessRequest("QueryPrepaidCards", params)
	if err != nil {
		log.Errorf("QueryPrepaidCards fail %s", err)
		return nil, err
	}
	cards := make([]SPrepaidCard, 0)
	err = body.Unmarshal(&cards, "Data", "PrepaidCard")
	if err != nil {
		log.Errorf("Unmarshal fail %s", err)
		return nil, err
	}
	return cards, nil
}