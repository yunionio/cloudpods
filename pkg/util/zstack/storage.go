package zstack

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

func (region *SRegion) getIStorages(zondId string) ([]cloudprovider.ICloudStorage, error) {
	primaryStorages, err := region.GetPrimaryStorages(zondId, "", "")
	if err != nil {
		return nil, err
	}
	istorage := []cloudprovider.ICloudStorage{}
	for _, primaryStorage := range primaryStorages {
		zone, err := region.GetZone(primaryStorage.ZoneUUID)
		if err != nil {
			return nil, err
		}
		switch primaryStorage.Type {
		case StorageTypeLocal:
			ilocalStorages, err := region.getILocalStorages(zone, primaryStorage.UUID, "")
			if err != nil {
				return nil, err
			}
			istorage = append(istorage, ilocalStorages...)
		case StorageTypeCeph:
			icephStorage, err := region.getICephStorages(zone, primaryStorage.UUID)
			if err != nil {
				return nil, err
			}
			istorage = append(istorage, icephStorage...)
		case StorageTypeVCenter:
		}
	}
	return istorage, nil
}
