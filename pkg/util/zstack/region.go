package zstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SRegion struct {
	client *SZStackClient

	Name string

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache
}

func (region *SRegion) GetClient() *SZStackClient {
	return region.client
}

func (region *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (region *SRegion) GetId() string {
	return region.Name
}

func (region *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_ZSTACK, region.Name)
}

func (region *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_ZSTACK, region.Name)
}

func (region *SRegion) IsEmulated() bool {
	return true
}

func (region *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_ZSTACK
}

func (region *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	return cloudprovider.SGeographicInfo{}
}

func (region *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (region *SRegion) Refresh() error {
	// do nothing
	return nil
}

func (region *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return region.GetHost("", id)
}

func (region *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return region.GetStorageWithZone(id)
}

func (region *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts, err := region.GetHosts("", "")
	if err != nil {
		return nil, err
	}
	ihosts := []cloudprovider.ICloudHost{}
	for i := 0; i < len(hosts); i++ {
		ihosts = append(ihosts, &hosts[i])
	}
	return ihosts, nil
}

func (region *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := region.GetStorages("", "", "")
	if err != nil {
		return nil, err
	}
	istorages := []cloudprovider.ICloudStorage{}
	for i := 0; i < len(storages); i++ {
		istorages = append(istorages, &storages[i])
	}
	return istorages, nil
}

func (region *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	caches, err := region.GetIStoragecaches()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(caches); i++ {
		if caches[i].GetGlobalId() == id {
			return caches[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	region.storageCache = &SStoragecache{region: region}
	return []cloudprovider.ICloudStoragecache{region.storageCache}, nil
}

func (region *SRegion) GetIVpcById(vpcId string) (cloudprovider.ICloudVpc, error) {
	return &SVpc{region: region}, nil
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

func (region *SRegion) GetZone(zoneId string) (*SZone, error) {
	zones, err := region.GetZones(zoneId)
	if err != nil {
		return nil, err
	}
	if len(zones) == 1 {
		if zones[0].UUID == zoneId {
			return &zones[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(zones) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetZones(zoneId string) ([]SZone, error) {
	zones := []SZone{}
	params := []string{}
	if len(zoneId) > 0 {
		params = append(params, "q=uuid="+zoneId)
	}
	err := region.client.listAll("zones", params, &zones)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(zones); i++ {
		zones[i].region = region
	}
	return zones, nil
}

func (region *SRegion) fetchZones() {
	if region.izones == nil || len(region.izones) == 0 {
		zones, err := region.GetZones("")
		if err != nil {
			log.Errorf("failed to get zones error: %v", err)
			return
		}
		region.izones = []cloudprovider.ICloudZone{}
		for i := 0; i < len(zones); i++ {
			region.izones = append(region.izones, &zones[i])
		}
	}
}

func (region *SRegion) fetchInfrastructure() error {
	region.fetchZones()
	region.GetIVpcs()
	return nil
}

func (region *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if region.izones == nil {
		if err := region.fetchInfrastructure(); err != nil {
			return nil, err
		}
	}
	return region.izones, nil
}

func (region *SRegion) GetVpc() *SVpc {
	return &SVpc{region: region}
}

func (region *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	region.ivpcs = []cloudprovider.ICloudVpc{region.GetVpc()}
	return region.ivpcs, nil
}

func (region *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (region *SRegion) CreateEIP(name string, bwMbps int, chargeType string, bgpType string) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	return region.GetEip(eipId)
}

func (region *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) DeleteSecurityGroup(vpcId, secGrpId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	eips, err := region.GetEips("", "")
	if err != nil {
		return nil, err
	}
	ieips := []cloudprovider.ICloudEIP{}
	for i := 0; i < len(eips); i++ {
		ieips = append(ieips, &eips[i])
	}
	return ieips, nil
}

func (region *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := region.GetSnapshots("", "")
	if err != nil {
		return nil, err
	}
	isnapshots := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = region
		isnapshots = append(isnapshots, &snapshots[i])
	}
	return isnapshots, nil
}

func (region *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return region.GetSnapshot(snapshotId)
}

func (region *SRegion) GetSkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	offerings, err := region.GetInstanceOfferings("")
	if err != nil {
		return nil, err
	}
	iskus := []cloudprovider.ICloudSku{}
	for i := 0; i < len(offerings); i++ {
		iskus = append(iskus, &offerings[i])
	}
	return iskus, nil
}

func (region *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}
