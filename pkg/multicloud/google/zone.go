// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package google

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SZone struct {
	region *SRegion

	Description           string
	ID                    string
	Kind                  string
	Name                  string
	Region                string
	SelfLink              string
	AvailableCpuPlatforms []string
	Status                string
}

func (region *SRegion) GetZone(id string) (*SZone, error) {
	zone := &SZone{}
	return zone, region.Get(id, zone)
}

func (region *SRegion) GetZones(regionId string, maxResults int, pageToken string) ([]SZone, error) {
	zones := []SZone{}
	params := map[string]string{}
	if len(regionId) > 0 {
		params["filter"] = fmt.Sprintf(`region="%s/regions/%s"`, region.GetUrlPrefixWithProjectId(), regionId)
	}
	resource := "zones"
	return zones, region.List(resource, params, maxResults, pageToken, &zones)
}

func (zone *SZone) GetName() string {
	return zone.Name
}

func (zone *SZone) GetGlobalId() string {
	return zone.GetId()
}

func (zone *SZone) GetId() string {
	return fmt.Sprintf("%s/%s", zone.region.GetGlobalId(), zone.Name)
}

func (zone *SZone) GetIHostById(hostId string) (cloudprovider.ICloudHost, error) {
	if hostId != zone.getHost().GetGlobalId() {
		return nil, cloudprovider.ErrNotFound
	}
	return &SHost{zone: zone}, nil
}

func (zone *SZone) getHost() *SHost {
	return &SHost{zone: zone}
}

func (zone *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	host := zone.getHost()
	return []cloudprovider.ICloudHost{host}, nil
}

func (zone *SZone) GetIRegion() cloudprovider.ICloudRegion {
	return zone.region
}

func (zone *SZone) GetIStorageById(storageId string) (cloudprovider.ICloudStorage, error) {
	storage, err := zone.region.GetStorage(storageId)
	if err != nil {
		return nil, err
	}
	storage.zone = zone
	return storage, nil
}

func (zone *SZone) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := zone.region.GetStorages(zone.Name, 0, "")
	if err != nil {
		return nil, err
	}
	istorages := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].zone = zone
		istorages = append(istorages, &storages[i])
	}
	return istorages, nil
}

func (zone *SZone) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (zone *SZone) IsEmulated() bool {
	return false
}

func (zone *SZone) Refresh() error {
	return nil
}

func (zone *SZone) GetStatus() string {
	if zone.Status == "UP" {
		return api.ZONE_ENABLE
	}
	return api.ZONE_SOLDOUT
}
