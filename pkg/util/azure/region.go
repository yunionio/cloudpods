package azure

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
)

type VMSize struct {
	MaxDataDiskCount     int32
	MemoryInMB           int32
	NumberOfCores        int32
	Name                 string
	OsDiskSizeInMB       int32
	ResourceDiskSizeInMB int32
}

type SRegion struct {
	client *SAzureClient

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	vmSize         map[string]VMSize
	ID             string
	SubscriptionID string
	Name           string
	DisplayName    string
	Latitude       string
	Longitude      string
}

/////////////////////////////////////////////////////////////////////////////
func (self *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (self *SRegion) GetClient() *SAzureClient {
	return self.client
}

func (self *SRegion) fetchVMSize() error {
	computeClient := compute.NewVirtualMachineSizesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if vmSizeList, err := computeClient.List(context.Background(), self.Name); err != nil {
		return err
	} else {
		self.vmSize = make(map[string]VMSize, len(*vmSizeList.Value))
		for _, _vmSize := range *vmSizeList.Value {
			vmSize := VMSize{}
			jsonutils.Update(&vmSize, _vmSize)
			self.vmSize[*_vmSize.Name] = vmSize
		}
	}
	return nil
}

func (self *SRegion) getVMSize(size string) (*VMSize, error) {
	if self.vmSize == nil || len(self.vmSize) == 0 {
		if err := self.fetchVMSize(); err != nil {
			return nil, err
		}
	}
	if vmSize, ok := self.vmSize[size]; !ok {
		return nil, cloudprovider.ErrNotFound
	} else {
		return &vmSize, nil
	}
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) GetId() string {
	return self.Name
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_AZURE_CN, self.DisplayName)
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_AZURE, self.Name)
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_AZURE
}

func (self *SRegion) GetLatitude() float32 {
	return 0.0
}

func (self *SRegion) GetLongitude() float32 {
	return 0.0
}

func (self *SRegion) GetStatus() string {
	return models.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	return nil, nil
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return nil, nil
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, nil
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return nil, nil
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return nil, nil
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	return nil, nil
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	return nil, nil
}

func (self *SRegion) fetchZones() error {
	if self.izones == nil {
		self.izones = make([]cloudprovider.ICloudZone, 1)
		zone := SZone{region: self, Name: self.Name}
		self.izones[0] = &zone
	}
	return nil
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if self.izones == nil {
		if err := self.fetchInfrastructure(); err != nil {
			return nil, err
		}
	}
	return self.izones, nil
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (self *SRegion) getStorage() ([]SStorage, error) {
	storages := make([]SStorage, 0)
	storageClient := storage.NewAccountsClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	storageClient.Authorizer = self.client.authorizer
	if storageList, err := storageClient.List(context.Background()); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&storages, storageList.Value); err != nil {
		return storages, err
	}
	return storages, nil
}

func (self *SRegion) getVpcs() ([]SVpc, error) {
	vpcs := make([]SVpc, 0)
	networkClient := network.NewVirtualNetworksClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	networkClient.Authorizer = self.client.authorizer
	if vpcList, err := networkClient.ListAll(context.Background()); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&vpcs, vpcList.Values()); err != nil {
		return nil, err
	}
	return vpcs, nil
}

func (self *SRegion) fetchIVpc() error {
	if vpcs, err := self.getVpcs(); err != nil {
		return err
	} else {
		self.ivpcs = make([]cloudprovider.ICloudVpc, 0)
		for i := 0; i < len(vpcs); i++ {
			if vpcs[i].Location == self.Name {
				vpcs[i].region = self
				self.ivpcs = append(self.ivpcs, &vpcs[i])
			}
		}
	}
	return nil
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if self.ivpcs == nil {
		if err := self.fetchInfrastructure(); err != nil {
			return nil, err
		}
	}
	for _, vpc := range self.ivpcs {
		log.Debugf("find vpc %s for region %s", vpc.GetName(), self.GetName())
	}
	return self.ivpcs, nil
}

func (self *SRegion) fetchInfrastructure() error {
	if err := self.fetchZones(); err != nil {
		return err
	}
	if err := self.fetchIVpc(); err != nil {
		return err
	}
	for i := 0; i < len(self.ivpcs); i++ {
		for j := 0; j < len(self.izones); j++ {
			zone := self.izones[j].(*SZone)
			vpc := self.ivpcs[i].(*SVpc)
			wire := SWire{zone: zone, vpc: vpc}
			zone.addWire(&wire)
			vpc.addWire(&wire)
		}
	}
	return nil
}
