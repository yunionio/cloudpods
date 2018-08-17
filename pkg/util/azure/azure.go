package azure

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/services/preview/subscription/mgmt/2018-03-01-preview/subscription"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	CLOUD_PROVIDER_AZURE    = models.CLOUD_PROVIDER_AZURE
	CLOUD_PROVIDER_AZURE_CN = "微软"

	AZURE_DEFAULT_ENVIRONMENT = "AzureChinaCloud"
	AZURE_DEFAULT_APPLICATION = "Azure-Yunion-API"

	AZURE_API_VERSION = "2014-05-26"
)

type SAzureClient struct {
	providerId     string
	providerName   string
	subscriptionId string
	tenantId       string
	clientId       string
	clientScret    string
	baseUrl        string
	secret         string
	resourceGroups map[string][]string
	authorizer     autorest.Authorizer
	iregions       []cloudprovider.ICloudRegion
}

func NewAzureClient(providerId string, providerName string, accessKey string, secret string, url string) (*SAzureClient, error) {
	url = strings.Replace(url, "login", "management", -1)
	if clientInfo := strings.Split(secret, "/"); len(clientInfo) == 3 {
		client := SAzureClient{providerId: providerId, providerName: providerName, tenantId: accessKey, secret: secret, baseUrl: url}
		client.clientId, client.clientScret, client.subscriptionId = clientInfo[0], clientInfo[1], clientInfo[2]
		if err := client.fetchAzureInof(); err != nil {
			return nil, err
		} else if err := client.fetchRegions(); err != nil {
			return nil, err
		}
		return &client, nil
	} else {
		return nil, httperrors.NewUnauthorizedError("clientId、clientScret or subscriptId input error")
	}
}

func (self *SAzureClient) fetchAzureInof() error {
	conf := auth.NewClientCredentialsConfig(self.clientId, self.clientScret, self.tenantId)
	conf.Resource = self.baseUrl
	conf.AADEndpoint = strings.Replace(self.baseUrl, "management", "login", -1)
	if authorizer, err := conf.Authorizer(); err != nil {
		return err
	} else {
		self.authorizer = authorizer
	}
	resourceClient := resources.NewClientWithBaseURI(self.baseUrl, self.subscriptionId)
	resourceClient.Authorizer = self.authorizer
	if resourceList, err := resourceClient.List(context.Background(), "", "", nil); err != nil {
		return err
	} else {
		self.resourceGroups = make(map[string][]string, len(resourceList.Values()))
		for _, resource := range resourceList.Values() {
			if _, ok := self.resourceGroups[*resource.Type]; !ok {
				self.resourceGroups[*resource.Type] = []string{}
			}
			self.resourceGroups[*resource.Type] = append(self.resourceGroups[*resource.Type], *resource.Name)
			log.Errorf("find resource group: %s => %s", *resource.Type, *resource.Name)
		}
	}

	return nil
}

func (self *SAzureClient) UpdateAccount(tenantId, secret string) error {
	if self.tenantId != tenantId || self.secret != secret {
		self.tenantId = tenantId
		self.secret = secret
		if clientInfo := strings.Split(secret, "/"); len(clientInfo) == 3 {
			self.clientId, self.clientScret, self.subscriptionId = clientInfo[0], clientInfo[1], clientInfo[2]
			conf := auth.NewClientCredentialsConfig(self.clientId, self.clientScret, self.tenantId)
			conf.Resource = self.baseUrl
			conf.AADEndpoint = strings.Replace(self.baseUrl, "management", "login", -1)
			if authorizer, err := conf.Authorizer(); err != nil {
				return err
			} else {
				self.authorizer = authorizer
			}
		} else {
			return httperrors.NewUnauthorizedError("clientId、clientScret or subscriptId input error")
		}
		return self.fetchAzureInof()
	} else {
		return nil
	}
}

func (self *SAzureClient) fetchRegions() error {
	locationClient := subscription.NewSubscriptionsClientWithBaseURI(self.baseUrl)
	locationClient.Authorizer = self.authorizer
	if locationList, err := locationClient.ListLocations(context.Background(), self.subscriptionId); err != nil {
		return err
	} else {
		regions := make([]SRegion, len(*locationList.Value))
		self.iregions = make([]cloudprovider.ICloudRegion, len(regions))
		for i, location := range *locationList.Value {
			region := SRegion{SubscriptionID: self.subscriptionId}
			if err := jsonutils.Update(&region, location); err != nil {
				return err
			}
			region.client = self
			self.iregions[i] = &region
			log.Infof("find region: %s", jsonutils.Marshal(region).PrettyString())
		}
	}
	return nil
}

func (self *SAzureClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SAzureClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SAzureClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SAzureClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
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

func (self *SAzureClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
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

func (self *SAzureClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
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

func (self *SAzureClient) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
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
