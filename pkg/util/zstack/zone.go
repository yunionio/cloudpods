package zstack

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SZone struct {
	region *SRegion

	ZStackBasic
	Type  string
	State string

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	ihosts []cloudprovider.ICloudHost
}

func (zone *SZone) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (zone *SZone) GetId() string {
	return zone.Name
}

func (zone *SZone) GetName() string {
	return zone.Name
}

func (zone *SZone) GetGlobalId() string {
	return zone.GetId()
}

func (zone *SZone) IsEmulated() bool {
	return false
}

func (zone *SZone) GetStatus() string {
	if zone.State == "Enabled" {
		return api.ZONE_ENABLE
	}
	return api.ZONE_DISABLE
}

func (zone *SZone) Refresh() error {
	// do nothing
	return nil
}

func (zone *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return zone.region
}

func (region *SRegion) GetStorageWithZone(storageId string) (*SStorage, error) {
	storage, err := region.GetStorage(storageId)
	if err != nil {
		log.Errorf("failed to found storage %s error: %v", storageId, err)
		return nil, err
	}
	zone, err := region.GetZone(storage.ZoneUUID)
	if err != nil {
		return nil, err
	}
	storage.zone = zone
	return storage, nil
}

func (region *SRegion) GetStorage(storageId string) (*SStorage, error) {
	storages, err := region.GetStorages("", "", storageId)
	if err != nil {
		return nil, err
	}
	if len(storages) == 1 {
		if storages[0].UUID == storageId {
			return &storages[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(storages) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetStorages(zoneId, clusterId, storageId string) ([]SStorage, error) {
	storages := []SStorage{}
	params := []string{}
	if len(zoneId) > 0 {
		params = append(params, "q=zone.uuid="+zoneId)
	}
	if len(clusterId) > 0 {
		params = append(params, "q=cluster.uuid="+clusterId)
	}
	if SkipEsxi {
		params = append(params, "q=type!=VCenter")
	}
	if len(storageId) > 0 {
		params = append(params, "q=uuid="+storageId)
	}
	return storages, region.client.listAll("primary-storage", params, &storages)
}

func (zone *SZone) fetchStorages(clusterId string) error {
	storages, err := zone.region.GetStorages(zone.UUID, clusterId, "")
	if err != nil {
		return err
	}
	zone.istorages = []cloudprovider.ICloudStorage{}
	for i := 0; i < len(storages); i++ {
		storages[i].zone = zone
		zone.istorages = append(zone.istorages, &storages[i])
	}
	return nil
}

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	if zone.istorages == nil || len(zone.istorages) == 0 {
		return zone.istorages, zone.fetchStorages("")
	}
	return zone.istorages, nil
}

func (zone *SZone) GetIStorageById(storageId string) (cloudprovider.ICloudStorage, error) {
	storages, err := zone.region.GetStorages(zone.UUID, "", storageId)
	if err != nil {
		return nil, err
	}
	if len(storages) == 1 {
		if storages[0].UUID == storageId {
			storages[0].zone = zone
			return &storages[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(storages) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetHosts(zoneId string, hostId string) ([]SHost, error) {
	hosts := []SHost{}
	params := []string{}
	if len(zoneId) > 0 {
		params = append(params, "q=zone.uuid="+zoneId)
	}
	if len(hostId) > 0 {
		params = append(params, "q=uuid="+hostId)
	}
	if SkipEsxi {
		params = append(params, "q=hypervisorType!=ESX")
	}
	return hosts, region.client.listAll("hosts", params, &hosts)
}

func (region *SRegion) GetHost(zoneId string, hostId string) (*SHost, error) {
	hosts, err := region.GetHosts(zoneId, hostId)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 1 {
		if hosts[0].UUID == hostId {
			return &hosts[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(hosts) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (zone *SZone) GetIHostById(hostId string) (cloudprovider.ICloudHost, error) {
	host, err := zone.region.GetHost(zone.UUID, hostId)
	if err != nil {
		return nil, err
	}
	host.zone = zone
	return host, nil
}

func (zone *SZone) fetchHosts() error {
	hosts, err := zone.region.GetHosts(zone.UUID, "")
	if err != nil {
		return err
	}
	zone.ihosts = []cloudprovider.ICloudHost{}
	for i := 0; i < len(hosts); i++ {
		hosts[i].zone = zone
		zone.ihosts = append(zone.ihosts, &hosts[i])
	}
	return nil
}

func (zone *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	if zone.ihosts == nil || len(zone.ihosts) == 0 {
		return zone.ihosts, zone.fetchHosts()
	}
	return zone.ihosts, nil
}

func (zone *SZone) GetIWires() ([]cloudprovider.ICloudWire, error) {
	if zone.iwires == nil || len(zone.iwires) == 0 {
		wires, err := zone.region.GetWires(zone.UUID, "", "")
		if err != nil {
			return nil, err
		}
		zone.iwires = []cloudprovider.ICloudWire{}
		for i := 0; i < len(wires); i++ {
			zone.iwires = append(zone.iwires, &wires[i])
		}
	}
	return zone.iwires, nil
}
