package provider

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
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

func (self *SOpenStackProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	projectName, _ := data.GetString("project_name")
	if len(projectName) == 0 {
		return httperrors.NewMissingParameterError("project_name")
	}
	username, _ := data.GetString("username")
	if len(username) == 0 {
		return httperrors.NewMissingParameterError("username")
	}
	password, _ := data.GetString("password")
	if len(password) == 0 {
		return httperrors.NewMissingParameterError("password")
	}
	authURL, _ := data.GetString("auth_url")
	if len(authURL) == 0 {
		return httperrors.NewMissingParameterError("auth_url")
	}
	data.Set("account", jsonutils.NewString(fmt.Sprintf("%s/%s", projectName, username)))
	data.Set("secret", jsonutils.NewString(password))
	data.Set("access_url", jsonutils.NewString(authURL))
	return nil
}

func (self *SOpenStackProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	projectName, _ := data.GetString("project_name")
	if len(projectName) == 0 {
		accountInfo := strings.Split(cloudaccount, "/")
		if len(accountInfo) < 2 {
			return nil, httperrors.NewMissingParameterError("project_name")
		}
		projectName = accountInfo[1]
	}
	username, _ := data.GetString("username")
	if len(username) == 0 {
		return nil, httperrors.NewMissingParameterError("username")
	}
	password, _ := data.GetString("password")
	if len(password) == 0 {
		return nil, httperrors.NewMissingParameterError("password")
	}
	account := &cloudprovider.SCloudaccount{
		Account: fmt.Sprintf("%s/%s", projectName, username),
		Secret:  password,
	}
	return account, nil
}

func (self *SOpenStackProviderFactory) GetProvider(providerId, providerName, url, account, password string) (cloudprovider.ICloudProvider, error) {
	accountInfo := strings.Split(account, "/")
	if len(accountInfo) < 2 {
		return nil, fmt.Errorf("Missing username or project name %s", account)
	}
	project, username := accountInfo[0], accountInfo[1]
	client, err := openstack.NewOpenStackClient(providerId, providerName, url, username, password, project)
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

func (self *SOpenStackProvider) GetVersion() string {
	return ""
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
	return jsonutils.NewDict(), nil
}

func (self *SOpenStackProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SOpenStackProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return self.client.GetIRegions()
}

func (self *SOpenStackProvider) GetIRegionById(extId string) (cloudprovider.ICloudRegion, error) {
	return self.client.GetIRegionById(extId)
}

func (self *SOpenStackProvider) GetBalance() (float64, error) {
	return 0.0, nil
}

func (self *SOpenStackProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}
