package openstack

import (
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/secrules"
)

type SRegion struct {
	client *SOpenStackClient

	Name string

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	//storageCache *SStoragecache
}

func (region *SRegion) GetClient() *SOpenStackClient {
	return region.client
}

func (region *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (region *SRegion) GetId() string {
	return region.Name
}

func (region *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_OPENSTACK, region.Name)
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_OPENSTACK, region.Name)
}

func (region *SRegion) IsEmulated() bool {
	return false
}

func (region *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_OPENSTACK
}

func (region *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (region *SRegion) GetStatus() string {
	return models.CLOUD_REGION_STATUS_INSERVER
}

func (region *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (region *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	// izones, err := self.GetIZones()
	// if err != nil {
	// 	return nil, err
	// }
	// for i := 0; i < len(izones); i++ {
	// 	ihost, err := izones[i].GetIHostById(id)
	// 	if err == nil {
	// 		return ihost, nil
	// 	} else if err != cloudprovider.ErrNotFound {
	// 		return nil, err
	// 	}
	// }
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	// izones, err := self.GetIZones()
	// if err != nil {
	// 	return nil, err
	// }
	// for i := 0; i < len(izones); i += 1 {
	// 	istore, err := izones[i].GetIStorageById(id)
	// 	if err == nil {
	// 		return istore, nil
	// 	} else if err != cloudprovider.ErrNotFound {
	// 		return nil, err
	// 	}
	// }
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	iHosts := []cloudprovider.ICloudHost{}
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i++ {
		iZoneHost, err := izones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		iHosts = append(iHosts, iZoneHost...)
	}
	return iHosts, nil
}

func (region *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	// izones, err := self.GetIZones()
	// if err != nil {
	// 	return nil, err
	// }
	// for i := 0; i < len(izones); i += 1 {
	// 	iZoneStores, err := izones[i].GetIStorages()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	iStores = append(iStores, iZoneStores...)
	// }
	return iStores, nil
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	// storageCache := self.getStoragecache()
	// if storageCache.GetGlobalId() == id {
	// 	return self.storageCache, nil
	// }
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	// ivpcs, err := self.GetIVpcs()
	// if err != nil {
	// 	return nil, err
	// }
	// for i := 0; i < len(ivpcs); i++ {
	// 	if ivpcs[i].GetGlobalId() == id {
	// 		return ivpcs[i], nil
	// 	}
	// }
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i++ {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) fetchZones() error {
	zones, err := region.GetIZones()
	if err != nil {
		return err
	}
	region.izones = zones
	return nil
}

func (region *SRegion) fetchInfrastructure() error {
	if err := region.fetchZones(); err != nil {
		return err
	}
	// err = region.fetchIVpc()
	// if err != nil {
	// 	return err
	// }
	// for i := 0; i < len(region.ivpcs); i++ {
	// 	for j := 0; j < len(region.izones); j++ {
	// 		zone := region.izones[j].(*SZone)
	// 		vpc := region.ivpcs[i].(*SVpc)
	// 		wire := SWire{zone: zone, vpc: vpc}
	// 		zone.addWire(&wire)
	// 		vpc.addWire(&wire)
	// 	}
	// }
	return nil
}

func (region *SRegion) Get(url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return region.client.Get(region.Name, url, microversion, body)
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	_, resp, err := region.Get("/os-availability-zone", "", jsonutils.NewDict())
	if err != nil {
		return nil, err
	}
	zones := []SZone{}
	if err := resp.Unmarshal(&zones, "availabilityZoneInfo"); err != nil {
		return nil, err
	}
	izones := []cloudprovider.ICloudZone{}
	for i := 0; i < len(zones); i++ {
		if zones[i].ZoneName == "internal" {
			continue
		}
		zones[i].region = region
		izones = append(izones, &zones[i])
	}
	return izones, nil
}

func (region *SRegion) GetVersion(service string) (string, string, error) {
	return region.client.getComputeVersion(region.Name, service)
}

func (region *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	// if self.ivpcs == nil || self.iclassicVpcs == nil {
	// 	if err := self.fetchInfrastructure(); err != nil {
	// 		return nil, err
	// 	}
	// }
	// for _, vpc := range self.ivpcs {
	// 	log.Debugf("find vpc %s for region %s", vpc.GetName(), self.GetName())
	// }
	// for _, vpc := range self.iclassicVpcs {
	// 	log.Debugf("find classic vpc %s for region %s", vpc.GetName(), self.GetName())
	// }
	// ivpcs := self.ivpcs
	// if len(self.iclassicVpcs) > 0 {
	// 	ivpcs = append(ivpcs, self.iclassicVpcs...)
	// }
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (region *SRegion) DeleteSecurityGroup(vpcId, secgroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) SyncSecurityGroup(secgroupId, vpcId, name, desc string, rules []secrules.SecurityRule) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateEIP(name string, bwMbps int, chargeType string) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (region *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (region *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadbalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadbalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}
