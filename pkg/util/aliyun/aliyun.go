package aliyun

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"

	"github.com/yunionio/onecloud/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_ALIYUN    = "Aliyun"
	CLOUD_PROVIDER_ALIYUN_CN = "阿里云"

	ALIYUN_DEFAULT_REGION = "cn-hangzhou"
)

type SAliyunClient struct {
	providerId string
	accessKey  string
	secret     string
	regions    []SRegion
	iregions   []cloudprovider.ICloudRegion
}

func NewAliyunClient(providerId string, accessKey string, secret string) (*SAliyunClient, error) {
	client := SAliyunClient{providerId: providerId, accessKey: accessKey, secret: secret}
	err := client.fetchRegions()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func jsonRequest(client *sdk.Client, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	req := requests.NewCommonRequest()
	req.Domain = "ecs.aliyuncs.com"
	req.Version = "2014-05-26"
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

func (self *SAliyunClient) fetchRegions() error {
	body, err := self.jsonRequest("DescribeRegions", nil)
	if err != nil {
		return err
	}

	self.regions = make([]SRegion, 0)
	err = body.Unmarshal(&self.regions, "Regions", "Region")
	if err != nil {
		log.Errorf("unmarshal json error %s", err)
		return err
	}
	self.iregions = make([]cloudprovider.ICloudRegion, len(self.regions))
	for i := 0; i < len(self.regions); i += 1 {
		self.regions[i].client = self
		self.iregions[i] = &self.regions[i]
	}
	return nil
}

func (self *SAliyunClient) GetRegions() []SRegion {
	return self.regions
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
	for i := 0; i < len(self.regions); i += 1 {
		if self.regions[i].RegionId == regionId {
			return &self.regions[i]
		}
	}
	return nil
}
