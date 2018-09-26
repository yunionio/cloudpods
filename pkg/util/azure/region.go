package azure

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/seclib2"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
)

type SVMSize struct {
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

	ID             string
	SubscriptionID string
	Name           string
	DisplayName    string
	Latitude       float32
	Longitude      float32
}

/////////////////////////////////////////////////////////////////////////////
func (self *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (self *SRegion) GetClient() *SAzureClient {
	return self.client
}

func (self *SRegion) GetVMSize() (map[string]SVMSize, error) {
	computeClient := compute.NewVirtualMachineSizesClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	computeClient.Authorizer = self.client.authorizer
	if vmSizeList, err := computeClient.List(context.Background(), self.Name); err != nil {
		return nil, err
	} else {
		vmSizes := make(map[string]SVMSize, len(*vmSizeList.Value))
		for _, _vmSize := range *vmSizeList.Value {
			vmSize := SVMSize{}
			if err := jsonutils.Update(&vmSize, _vmSize); err != nil {
				return nil, err
			} else {
				vmSizes[*_vmSize.Name] = vmSize
			}
		}
		return vmSizes, nil
	}
}

func (self *SRegion) getHardwareProfile(cpu, memMB int) []string {
	if vmSizes, err := self.GetVMSize(); err != nil {
		return []string{}
	} else {
		profiles := make([]string, 0)
		for vmSize, info := range vmSizes {
			if info.MemoryInMB == int32(memMB) && info.NumberOfCores == int32(cpu) {
				profiles = append(profiles, vmSize)
			}
		}
		return profiles
	}
}

func (self *SRegion) getVMSize(size string) (*SVMSize, error) {
	if vmSizes, err := self.GetVMSize(); err != nil {
		return nil, err
	} else if vmSize, ok := vmSizes[size]; !ok {
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
	return self.Latitude
}

func (self *SRegion) GetLongitude() float32 {
	return self.Longitude
}

func (self *SRegion) GetStatus() string {
	return models.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	vpcClient := network.NewVirtualNetworksClientWithBaseURI(self.client.baseUrl, self.client.subscriptionId)
	vpcClient.Authorizer = self.client.authorizer
	addressPrefixes := []string{cidr}
	addressSpace := network.AddressSpace{AddressPrefixes: &addressPrefixes}
	properties := network.VirtualNetworkPropertiesFormat{AddressSpace: &addressSpace}
	parameters := network.VirtualNetwork{Name: &name, Location: &self.Name, VirtualNetworkPropertiesFormat: &properties}
	vpcId, resourceGroup, vpcName := pareResourceGroupWithName(name, VPC_RESOURCE)
	self.CreateResourceGroup(resourceGroup)
	if result, err := vpcClient.CreateOrUpdate(context.Background(), resourceGroup, vpcName, parameters); err != nil {
		return nil, err
	} else if err := result.WaitForCompletion(context.Background(), vpcClient.Client); err != nil {
		return nil, err
	} else if err := self.fetchInfrastructure(); err != nil {
		return nil, err
	} else if vpc, err := self.GetIVpcById(vpcId); err != nil {
		return nil, err
	} else {
		return vpc, nil
	}
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		ihost, err := izones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		istore, err := izones[i].GetIStorageById(id)
		if err == nil {
			return istore, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return self.storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	if ivpcs, err := self.GetIVpcs(); err != nil {
		return nil, err
	} else {
		_, resourceGroup, vpcName := pareResourceGroupWithName(id, VPC_RESOURCE)
		for i := 0; i < len(ivpcs); i++ {
			vpcId := ivpcs[i].GetId()
			_, _resourceGroup, _vpcName := pareResourceGroupWithName(vpcId, VPC_RESOURCE)
			if _resourceGroup == resourceGroup && _vpcName == vpcName {
				return ivpcs[i], nil
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	if izones, err := self.GetIZones(); err != nil {
		return nil, err
	} else {
		for i := 0; i < len(izones); i += 1 {
			if izones[i].GetGlobalId() == id {
				return izones[i], nil
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) getZoneById(id string) (*SZone, error) {
	if izones, err := self.GetIZones(); err != nil {
		return nil, err
	} else {
		for i := 0; i < len(izones); i += 1 {
			zone := izones[i].(*SZone)
			if zone.GetId() == id {
				return zone, nil
			}
		}
	}
	return nil, fmt.Errorf("no such zone %s", id)
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

func (self *SRegion) CreateInstanceSimple(name string, imgId string, cpu int, memGB int, storageType string, dataDiskSizesGB []int, networkId string, passwd string, publicKey string) (*SInstance, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		z := izones[i].(*SZone)
		log.Debugf("Search in zone %s", z.Name)
		net := z.getNetworkById(networkId)
		if net != nil {
			passwd := seclib2.RandomPassword2(12)
			inst, err := z.getHost().CreateVM(name, imgId, 30, cpu, memGB*1024, networkId, "", "", passwd, storageType, dataDiskSizesGB, publicKey, "")
			if err != nil {
				return nil, err
			}
			return inst.(*SInstance), nil
		}
	}
	return nil, fmt.Errorf("cannot find network %s", networkId)
}
