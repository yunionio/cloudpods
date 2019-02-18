package provider

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/esxi"
)

type SESXiProviderFactory struct {
}

func (self *SESXiProviderFactory) GetId() string {
	return esxi.CLOUD_PROVIDER_VMWARE
}

func (self *SESXiProviderFactory) GetName() string {
	return esxi.CLOUD_PROVIDER_VMWARE
}

func (self *SESXiProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", esxi.CLOUD_PROVIDER_VMWARE)
}

func (self *SESXiProviderFactory) IsPublicCloud() bool {
	return false
}

func (self *SESXiProviderFactory) IsOnPremise() bool {
	return true
}

func (self *SESXiProviderFactory) IsSupportPrepaidResources() bool {
	return false
}

func (self *SESXiProviderFactory) ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error {
	username, _ := data.GetString("username")
	if len(username) == 0 {
		return httperrors.NewMissingParameterError("username")
	}
	password, _ := data.GetString("password")
	if len(password) == 0 {
		return httperrors.NewMissingParameterError("password")
	}
	host, _ := data.GetString("host")
	if len(host) == 0 {
		return httperrors.NewMissingParameterError("host")
	}
	port, _ := data.Int("port")
	accessURL := fmt.Sprintf("https://%s:%d/sdk", host, port)
	if port == 0 || port == 443 {
		accessURL = fmt.Sprintf("https://%s/sdk", host)
	}
	data.Set("account", jsonutils.NewString(username))
	data.Set("secret", jsonutils.NewString(password))
	data.Set("access_url", jsonutils.NewString(accessURL))
	return nil
}

func (self *SESXiProviderFactory) ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*cloudprovider.SCloudaccount, error) {
	username, _ := data.GetString("username")
	if len(username) == 0 {
		return nil, httperrors.NewMissingParameterError("username")
	}
	password, _ := data.GetString("password")
	if len(password) == 0 {
		return nil, httperrors.NewMissingParameterError("password")
	}
	account := &cloudprovider.SCloudaccount{
		Account: username,
		Secret:  password,
	}
	return account, nil
}

func parseHostPort(host string, defPort int) (string, int, error) {
	colonPos := strings.IndexByte(host, ':')
	if colonPos > 0 {
		h := host[:colonPos]
		p, err := strconv.Atoi(host[colonPos+1:])
		if err != nil {
			log.Errorf("Invalid host %s", host)
			return "", 0, err
		}
		if p == 0 {
			p = defPort
		}
		return h, p, nil
	} else {
		return host, defPort, nil
	}
}

func (self *SESXiProviderFactory) GetProvider(providerId, providerName, urlStr, account, secret string) (cloudprovider.ICloudProvider, error) {
	parts, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	host, port, err := parseHostPort(parts.Host, 443)
	if err != nil {
		return nil, err
	}

	client, err := esxi.NewESXiClient(providerId, providerName, host, port, account, secret)
	if err != nil {
		return nil, err
	}
	return &SESXiProvider{
		SBaseProvider: cloudprovider.NewBaseProvider(self),
		client:        client,
	}, nil
}

func init() {
	factory := SESXiProviderFactory{}
	cloudprovider.RegisterFactory(&factory)
}

type SESXiProvider struct {
	cloudprovider.SBaseProvider
	client *esxi.SESXiClient
}

func (self *SESXiProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return self.client.About(), nil
}

func (self *SESXiProvider) GetVersion() string {
	return self.client.GetVersion()
}

func (self *SESXiProvider) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return self.client.GetSubAccounts()
}

func (self *SESXiProvider) GetIRegions() []cloudprovider.ICloudRegion {
	return nil
}

func (self *SESXiProvider) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SESXiProvider) GetBalance() (float64, error) {
	return 0.0, nil
}

func (self *SESXiProvider) GetOnPremiseIRegion() (cloudprovider.ICloudRegion, error) {
	return self.client, nil
}
