package zstack

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SZone struct {
	region *SRegion

	Name        string
	UUID        string
	Description string
	Type        string
	State       string

	iwires    []cloudprovider.ICloudWire
	istorages []cloudprovider.ICloudStorage

	//storageTypes []string
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
		return models.ZONE_ENABLE
	}
	return models.ZONE_DISABLE
}

func (zone *SZone) Refresh() error {
	// do nothing
	return nil
}

func (zone *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return zone.region
}

func (zone *SZone) getStorages(clusterUUID string) ([]SStorage, error) {
	storages := []SStorage{}
	params := []string{"q=zone.uuid=" + zone.UUID}
	if len(clusterUUID) > 0 {
		params = append(params, "q=cluster.uuid="+clusterUUID)
	}
	err := zone.region.client.listAll("primary-storage", params, &storages)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i++ {
		storages[i].zone = zone
	}
	return storages, nil
}

func (zone *SZone) fetchStorages(clusterUUID string) error {
	storages, err := zone.getStorages("")
	if err != nil {
		return err
	}
	zone.istorages = []cloudprovider.ICloudStorage{}
	for i := 0; i < len(storages); i++ {
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

func (zone *SZone) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotFound
}

func (zone *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	hosts := []SHost{}
	params := []string{"q=uuid=" + id, "q=zone.uuid=" + zone.UUID}
	err := zone.region.client.listAll("hosts", params, &hosts)
	if err != nil {
		return nil, err
	}
	count := len(hosts)
	switch count {
	case 0:
		return nil, cloudprovider.ErrNotFound
	case 1:
		hosts[0].zone = zone
		return &hosts[0], nil
	default:
		return nil, cloudprovider.ErrDuplicateId
	}
}

func (zone *SZone) fetchHosts() error {
	hosts := []SHost{}
	err := zone.region.client.listAll("hosts", []string{"q=zoneUuid=" + zone.UUID}, &hosts)
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
	return zone.iwires, nil
}
