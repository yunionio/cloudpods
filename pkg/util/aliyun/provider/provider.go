package provider

import (
	"github.com/yunionio/jsonutils"
	// "github.com/yunionio/log"

	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/util/aliyun"
)

type SAliyunProviderFactory struct {
	providerTable map[string]*SAliyunProvider
}

func (self *SAliyunProviderFactory) GetId() string {
	return aliyun.CLOUD_PROVIDER_ALIYUN
}

func (self *SAliyunProviderFactory) GetProvider(providerId, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	provider, ok := self.providerTable[providerId]
	if ok {
		return provider, nil
	}
	client, err := aliyun.NewAliyunClient(providerId, account, secret)
	if err != nil {
		return nil, err
	}
	self.providerTable[providerId] = &SAliyunProvider{client: client}
	return self.providerTable[providerId], nil
}

func init() {
	factory := SAliyunProviderFactory{
		providerTable: make(map[string]*SAliyunProvider),
	}
	cloudprovider.RegisterFactory(&factory)
}

type SAliyunProvider struct {
	client *aliyun.SAliyunClient
}

func (self *SAliyunProvider) IsPublicCloud() bool {
	return true
}

func (self *SAliyunProvider) GetId() string {
	return aliyun.CLOUD_PROVIDER_ALIYUN
}

func (self *SAliyunProvider) GetName() string {
	return aliyun.CLOUD_PROVIDER_ALIYUN_CN
}

func (self *SAliyunProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	return info, nil
}

func (self *SAliyunProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SAliyunProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}
