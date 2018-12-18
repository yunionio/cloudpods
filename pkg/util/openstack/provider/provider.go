package provider

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/openstack"
)

type SOpenStackProviderFactory struct {
	// providerTable map[string]*SOpenStackProvider
}

func (self *SOpenStackProviderFactory) GetId() string {
	return openstack.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return nil
}

func (self *SOpenStackProviderFactory) GetProvider(providerId, providerName, url, username, password string) (cloudprovider.ICloudProvider, error) {
	client, err := openstack.NewOpenStackClient(providerId, providerName, url, username, username, password)
	if err != nil {
		return nil, err
	}
	return &SOpenStackProvider{client: client}, nil
}

func init() {
	factory := SOpenStackProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SOpenStackProvider struct {
	client *openstack.SOpenStackClient
}

func (self *SOpenStackProvider) IsPublicCloud() bool {
	return false
}

func (self *SOpenStackProvider) IsOnPremiseInfrastructure() bool {
	return false
}

func (self *SOpenStackProvider) GetId() string {
	return openstack.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackProvider) GetName() string {
	return openstack.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return nil, nil
}

func (self *SOpenStackProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SOpenStackProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return nil
}

func (self *SOpenStackProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SOpenStackProvider) GetBalance() (float64, error) {
	return 0.0, nil
}

func (self *SOpenStackProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}
