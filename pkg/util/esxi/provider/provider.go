package provider

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/esxi"
)

type SESXiProviderFactory struct {
	providerTable map[string]*SESXiProvider
}

func (self *SESXiProviderFactory) GetId() string {
	return esxi.CLOUD_PROVIDER_VMWARE
}

func (self *SESXiProviderFactory) ValidateChangeBandwidth(instanceId string, bandwidth int64) error {
	return fmt.Errorf("Changing %s bandwidth is not supported", esxi.CLOUD_PROVIDER_VMWARE)
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
	provider, ok := self.providerTable[providerId]
	if ok {
		return provider, nil
	}
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
	self.providerTable[providerId] = &SESXiProvider{client: client}
	return self.providerTable[providerId], nil
}

func init() {
	factory := SESXiProviderFactory{
		providerTable: make(map[string]*SESXiProvider),
	}
	cloudprovider.RegisterFactory(&factory)
}

type SESXiProvider struct {
	client *esxi.SESXiClient
}

func (self *SESXiProvider) IsPublicCloud() bool {
	return false
}

func (self *SESXiProvider) GetId() string {
	return esxi.CLOUD_PROVIDER_VMWARE
}

func (self *SESXiProvider) GetName() string {
	return esxi.CLOUD_PROVIDER_VMWARE
}

func (self *SESXiProvider) GetSysInfo() (jsonutils.JSONObject, error) {
	return self.client.About(), nil
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

func (self *SESXiProvider) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host, err := self.client.FindHostByIp(id)
	if err != nil {
		return nil, err
	} else {
		return host, nil
	}
}

func (self *SESXiProvider) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SESXiProvider) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SESXiProvider) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SESXiProvider) GetBalance() (float64, error) {
	return 0.0, nil
}
