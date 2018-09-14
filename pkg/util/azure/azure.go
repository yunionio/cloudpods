package azure

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-03-31/consumption"
	"github.com/Azure/azure-sdk-for-go/services/preview/subscription/mgmt/2018-03-01-preview/subscription"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"

	"github.com/Azure/go-autorest/autorest"
	azureenv "github.com/Azure/go-autorest/autorest/azure"
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

	AZURE_API_VERSION = "2018-04-01"
)

const (
	DISK_RESOURCE     = "disk"
	INSTANCE_RESOURCE = "instance"
	VPC_RESOURCE      = "vpc"
	NIC_RESOURCE      = "nic"
	IMAGE_RESOURCE    = "image"
	STORAGE_RESOURCE  = "storage"
	SECGRP_RESOURCE   = "secgroup"
	EIP_RESOURCE      = "eip"
)

var defaultResourceGroups = map[string]string{
	DISK_RESOURCE:     "YunionDiskResource",
	INSTANCE_RESOURCE: "YunionInstanceResource",
	VPC_RESOURCE:      "YunionVpcResource",
	NIC_RESOURCE:      "YunionNicInterface",
	IMAGE_RESOURCE:    "YunionImageResource",
	STORAGE_RESOURCE:  "YunionStorageResource",
	SECGRP_RESOURCE:   "YunionSecgrpResource",
	EIP_RESOURCE:      "YunionEipResource",
}

type SAzureClient struct {
	providerId     string
	providerName   string
	subscriptionId string
	tenantId       string
	clientId       string
	clientScret    string
	baseUrl        string
	secret         string
	envName        string
	env            azureenv.Environment
	authorizer     autorest.Authorizer
	iregions       []cloudprovider.ICloudRegion
}

func NewAzureClient(providerId string, providerName string, accessKey string, secret string, envName string) (*SAzureClient, error) {
	if clientInfo, accountInfo := strings.Split(secret, "/"), strings.Split(accessKey, "/"); len(clientInfo) == 2 && len(accountInfo) == 2 {
		client := SAzureClient{providerId: providerId, providerName: providerName, secret: secret, envName: envName}
		client.clientId, client.clientScret = clientInfo[0], clientInfo[1]
		client.tenantId, client.subscriptionId = accountInfo[0], accountInfo[1]
		if env, err := azureenv.EnvironmentFromName(envName); err != nil {
			return nil, err
		} else {
			client.env = env
			client.baseUrl = env.ResourceManagerEndpoint
		}
		if err := client.fetchAzureInof(); err != nil {
			return nil, err
		} else if err := client.fetchRegions(); err != nil {
			return nil, err
		} else if err := client.fetchAzueResourceGroup(); err != nil {
			return nil, err
		}
		return &client, nil
	} else {
		return nil, httperrors.NewUnauthorizedError("clientId、clientScret or subscriptId input error")
	}
}

func pareResourceGroupWithName(s string, resourceType string) (string, string, string) {
	valid := regexp.MustCompile("resourceGroups/(.+)/providers/.+/(.+)$")
	if resourceGroups := valid.FindStringSubmatch(s); len(resourceGroups) == 3 {
		globalId := fmt.Sprintf("resourceGroups/%s/providers/%s/%s", resourceGroups[1], resourceType, resourceGroups[2])
		return globalId, resourceGroups[1], resourceGroups[2]
	}
	if len(s) == 0 {
		log.Errorf("pareResourceGroupWithName[%s] error", resourceType)
	}
	globalId := fmt.Sprintf("resourceGroups/%s/providers/%s/%s", defaultResourceGroups[resourceType], resourceType, s)
	return globalId, defaultResourceGroups[resourceType], s
}

func (self *SAzureClient) isResourceGroupExist(resourceGroup string) (bool, error) {
	groupClient := resources.NewGroupsClientWithBaseURI(self.baseUrl, self.subscriptionId)
	groupClient.Authorizer = self.authorizer
	if result, err := groupClient.CheckExistence(context.Background(), resourceGroup); err != nil {
		return false, err
	} else if result.StatusCode == 404 {
		return false, nil
	} else {
		return true, nil
	}
}

func (self *SAzureClient) createResourceGroup(resourceGruop string) error {
	groupClient := resources.NewGroupsClientWithBaseURI(self.baseUrl, self.subscriptionId)
	groupClient.Authorizer = self.authorizer
	region := self.iregions[0].(*SRegion)
	location := region.Name
	group := resources.Group{Location: &location}
	if _, err := groupClient.CreateOrUpdate(context.Background(), resourceGruop, group); err != nil {
		return err
	}
	return nil
}

func (self *SAzureClient) fetchAzueResourceGroup() error {
	for _, value := range defaultResourceGroups {
		if exist, err := self.isResourceGroupExist(value); err != nil {
			log.Errorf("Check ResourceGroup error: %v", err)
		} else if !exist {
			if err := self.createResourceGroup(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *SAzureClient) fetchAzureInof() error {
	conf := auth.NewClientCredentialsConfig(self.clientId, self.clientScret, self.tenantId)
	conf.Resource = self.env.ResourceManagerEndpoint
	conf.AADEndpoint = self.env.ActiveDirectoryEndpoint
	if authorizer, err := conf.Authorizer(); err != nil {
		return err
	} else {
		self.authorizer = authorizer
	}
	return nil
}

func (self *SAzureClient) UpdateAccount(tenantId, secret, envName string) error {
	if self.tenantId != tenantId || self.secret != secret || self.envName != envName {
		if env, err := azureenv.EnvironmentFromName(envName); err != nil {
			return err
		} else {
			self.env = env
			self.baseUrl = env.ResourceManagerEndpoint
		}
		if clientInfo, accountInfo := strings.Split(secret, "/"), strings.Split(tenantId, "/"); len(clientInfo) == 2 && len(accountInfo) == 2 {
			self.clientId, self.clientScret = clientInfo[0], clientInfo[1]
			self.tenantId, self.subscriptionId = accountInfo[0], accountInfo[1]
			conf := auth.NewClientCredentialsConfig(self.clientId, self.clientScret, self.tenantId)
			conf.Resource = self.env.ResourceManagerEndpoint
			conf.AADEndpoint = self.env.ActiveDirectoryEndpoint
			if authorizer, err := conf.Authorizer(); err != nil {
				return err
			} else {
				self.authorizer = authorizer
			}
			if err := self.fetchAzureInof(); err != nil {
				return err
			} else if err := self.fetchRegions(); err != nil {
				return err
			} else if err := self.fetchAzueResourceGroup(); err != nil {
				return err
			}
			return nil
		} else {
			return httperrors.NewUnauthorizedError("clientId、clientScret or subscriptId input error")
		}
	}
	return nil
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

type SAccountBalance struct {
	AvailableAmount     float64
	AvailableCashAmount float64
	CreditAmount        float64
	MybankCreditAmount  float64
	Currency            string
}

func (self *SAzureClient) QueryAccountBalance() (*SAccountBalance, error) {
	consumption.NewWithBaseURI(self.baseUrl, self.subscriptionId)
	costClient := consumption.NewWithBaseURI(self.baseUrl, self.subscriptionId)
	//costClient := costmanagement.NewBillingAccountDimensionsClientWithBaseURI(self.baseUrl, self.subscriptionId)
	costClient.Authorizer = self.authorizer
	if result, err := costClient.GetBalancesByBillingAccount(context.Background(), "quxuan@ioito.partner.onmschina.cn"); err != nil {
		//if result, err := costClient.Get(context.Background(), ""); err != nil {
		return nil, err
	} else {
		log.Errorf(jsonutils.Marshal(result).PrettyString())
	}
	return nil, nil
}
