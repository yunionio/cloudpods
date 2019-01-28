package provider

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/aws"
)

type SAwsProviderFactory struct {
}

func (self *SAwsProviderFactory) GetId() string {
	return aws.CLOUD_PROVIDER_AWS
}

func (self *SAwsProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return nil
}

func (self *SAwsProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	accessKeyID, _ := data.GetString("access_key_id")
	if len(accessKeyID) == 0 {
		return httperrors.NewMissingParameterError("access_key_id")
	}
	accessKeySecret, _ := data.GetString("access_key_secret")
	if len(accessKeySecret) == 0 {
		return httperrors.NewMissingParameterError("access_key_secret")
	}
	environment, _ := data.GetString("environment")
	if len(environment) == 0 {
		return httperrors.NewMissingParameterError("environment")
	}
	data.Set("account", jsonutils.NewString(accessKeyID))
	data.Set("secret", jsonutils.NewString(accessKeySecret))
	data.Set("access_url", jsonutils.NewString(environment))
	return nil
}

func (self *SAwsProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	accessKeyID, _ := data.GetString("access_key_id")
	if len(accessKeyID) == 0 {
		return nil, httperrors.NewMissingParameterError("access_key_id")
	}
	accessKeySecret, _ := data.GetString("access_key_secret")
	if len(accessKeySecret) == 0 {
		return nil, httperrors.NewMissingParameterError("access_key_secret")
	}
	account := &cloudprovider.SCloudaccount{
		Account: accessKeyID,
		Secret:  accessKeySecret,
	}
	return account, nil
}

func (self *SAwsProviderFactory) GetProvider(providerId, providerName, url, account, secret string) (cloudprovider.ICloudProvider, error) {
	client, err := aws.NewAwsClient(providerId, providerName, url, account, secret)
	if err != nil {
		return nil, err
	}
	return &SAwsProvider{client: client}, nil
}

func init() {
	factory := SAwsProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SAwsProvider struct {
	client *aws.SAwsClient
}

func (self *SAwsProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SAwsProvider) GetId() string {
	return aws.CLOUD_PROVIDER_AWS
}

func (self *SAwsProvider) GetName() string {
	return aws.CLOUD_PROVIDER_AWS_CN
}

func (self *SAwsProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SAwsProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	regions := self.client.GetIRegions()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewInt(int64(len(regions))), "region_count")
	info.Add(jsonutils.NewString(aws.AWS_API_VERSION), "api_version")
	return info, nil
}

func (self *SAwsProvider) GetVersion() string {
	return aws.AWS_API_VERSION
}

func (self *SAwsProvider) IsPublicCloud() bool {
	return true
}

func (self *SAwsProvider) IsOnPremiseInfrastructure() bool {
	return false
}

func (self *SAwsProvider) SyncSkuFromCloud() bool {
	return false
}

func (self *SAwsProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(id)
}

func (self *SAwsProvider) GetBalance() (float64, error) {
	balance, err := self.client.QueryAccountBalance()
	if err != nil {
		return 0.0, err
	}
	return balance.AvailableAmount, nil
}

func (self *SAwsProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SAwsProvider) SupportPrepaidResources() bool {
	return true
}
