package incloudsphere

import "yunion.io/x/onecloud/pkg/cloudprovider"

type SphereClient struct {
	*SphereClientConfig
}

type SphereClientConfig struct {
	cpcfg        cloudprovider.ProviderConfig
	accessKey    string
	accessSecret string
	host         string

	debug bool
}

func NewSphereClientConfig(host, accessKey, accessSecret string) *SphereClientConfig {
	return &SphereClientConfig{
		host:         host,
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
}

func (self *SphereClientConfig) Debug(debug bool) *SphereClientConfig {
	self.debug = debug
	return self
}

func (self *SphereClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *SphereClientConfig {
	self.cpcfg = cpcfg
	return self
}

func NewSphereClient(cfg *SphereClientConfig) (*SphereClient, error) {
	client := &SphereClient{
		SphereClientConfig: cfg,
	}
	return client, client.auth()
}

func (self *SphereClient) auth() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SphereClient) GetRegion() (*SRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SphereClient) GetRegions() ([]SRegion, error) {
	return nil, cloudprovider.ErrNotImplemented
}
