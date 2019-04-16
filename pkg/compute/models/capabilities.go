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

package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCapabilities struct {
	Hypervisors        []string `json:",allowempty"`
	ResourceTypes      []string `json:",allowempty"`
	StorageTypes       []string `json:",allowempty"`
	DataStorageTypes   []string `json:",allowempty"`
	GPUModels          []string `json:",allowempty"`
	MinNicCount        int
	MaxNicCount        int
	MinDataDiskCount   int
	MaxDataDiskCount   int
	SchedPolicySupport bool
	Usable             bool
	Specs              jsonutils.JSONObject
}

func GetCapabilities(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, region *SCloudregion, zone *SZone) (SCapabilities, error) {
	capa := SCapabilities{}
	capa.Hypervisors = getHypervisors(region, zone)
	capa.ResourceTypes = getResourceTypes(region, zone)
	capa.StorageTypes = getStorageTypes(region, zone, true)
	capa.DataStorageTypes = getStorageTypes(region, zone, false)
	capa.GPUModels = getGPUs(region, zone)
	capa.SchedPolicySupport = isSchedPolicySupported(region, zone)
	capa.MinNicCount = getMinNicCount(region, zone)
	capa.MaxNicCount = getMaxNicCount(region, zone)
	capa.MinDataDiskCount = getMinDataDiskCount(region, zone)
	capa.MaxDataDiskCount = getMaxDataDiskCount(region, zone)
	capa.Usable = isUsable(region, zone)
	if query == nil {
		query = jsonutils.NewDict()
	}
	var err error
	if region != nil {
		query.(*jsonutils.JSONDict).Add(jsonutils.NewString(region.GetId()), "region")
	}
	if zone != nil {
		query.(*jsonutils.JSONDict).Add(jsonutils.NewString(zone.GetId()), "zone")
	}
	mans := []ISpecModelManager{HostManager, IsolatedDeviceManager}
	capa.Specs, err = GetModelsSpecs(ctx, userCred, query.(*jsonutils.JSONDict), mans...)
	return capa, err
}

func getRegionZoneSubq(region *SCloudregion) *sqlchemy.SSubQuery {
	return ZoneManager.Query("id").Equals("cloudregion_id", region.GetId()).SubQuery()
}

func getHypervisors(region *SCloudregion, zone *SZone) []string {
	q := HostManager.Query("host_type", "manager_id")
	if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), subq))
	}
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	q = q.IsNotEmpty("host_type").IsNotNull("host_type")
	// q = q.Equals("host_status", HOST_ONLINE)
	q = q.IsTrue("enabled")
	q = q.Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	hypervisors := make([]string, 0)
	for rows.Next() {
		var hostType string
		var managerId string
		rows.Scan(&hostType, &managerId)
		if len(hostType) > 0 && IsProviderAccountEnabled(managerId) {
			hypervisor := api.HOSTTYPE_HYPERVISOR[hostType]
			if !utils.IsInStringArray(hypervisor, hypervisors) {
				hypervisors = append(hypervisors, hypervisor)
			}
		}
	}
	return hypervisors
}

func getResourceTypes(region *SCloudregion, zone *SZone) []string {
	q := HostManager.Query("resource_type", "manager_id")
	if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), subq))
	}
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	q = q.IsNotEmpty("resource_type").IsNotNull("resource_type")
	q = q.IsTrue("enabled")
	q = q.Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	resourceTypes := make([]string, 0)
	for rows.Next() {
		var resType string
		var managerId string
		rows.Scan(&resType, &managerId)
		if len(resType) > 0 && IsProviderAccountEnabled(managerId) {
			if !utils.IsInStringArray(resType, resourceTypes) {
				resourceTypes = append(resourceTypes, resType)
			}
		}
	}
	return resourceTypes
}

func getStorageTypes(region *SCloudregion, zone *SZone, isSysDisk bool) []string {
	storages := StorageManager.Query().SubQuery()
	hostStorages := HoststorageManager.Query().SubQuery()
	hosts := HostManager.Query().SubQuery()

	q := storages.Query(storages.Field("storage_type"), storages.Field("medium_type"))
	q = q.Join(hostStorages, sqlchemy.Equals(
		hostStorages.Field("storage_id"),
		storages.Field("id"),
	))
	q = q.Join(hosts, sqlchemy.Equals(
		hosts.Field("id"),
		hostStorages.Field("host_id"),
	))
	if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Filter(sqlchemy.In(storages.Field("zone_id"), subq))
	}
	if zone != nil {
		q = q.Filter(sqlchemy.Equals(storages.Field("zone_id"), zone.Id))
	}
	q = q.Filter(sqlchemy.Equals(hosts.Field("resource_type"), api.HostResourceTypeShared))
	q = q.Filter(sqlchemy.IsNotEmpty(storages.Field("storage_type")))
	q = q.Filter(sqlchemy.IsNotNull(storages.Field("storage_type")))
	q = q.Filter(sqlchemy.IsNotEmpty(storages.Field("medium_type")))
	q = q.Filter(sqlchemy.IsNotNull(storages.Field("medium_type")))
	q = q.Filter(sqlchemy.In(storages.Field("status"), []string{api.STORAGE_ENABLED, api.STORAGE_ONLINE}))
	q = q.Filter(sqlchemy.IsTrue(storages.Field("enabled")))
	if isSysDisk {
		q = q.Filter(sqlchemy.IsTrue(storages.Field("is_sys_disk_store")))
	}
	q = q.Filter(sqlchemy.NotEquals(hosts.Field("host_type"), api.HOST_TYPE_BAREMETAL))
	q = q.Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	storageTypes := make([]string, 0)
	for rows.Next() {
		var storageType, mediumType string
		rows.Scan(&storageType, &mediumType)
		if len(storageType) > 0 && len(mediumType) > 0 {
			storageTypes = append(storageTypes, fmt.Sprintf("%s/%s", storageType, mediumType))
		}
	}
	return storageTypes
}

func getGPUs(region *SCloudregion, zone *SZone) []string {
	devices := IsolatedDeviceManager.Query().SubQuery()
	hosts := HostManager.Query().SubQuery()

	q := devices.Query(devices.Field("model"))
	if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Join(hosts, sqlchemy.Equals(devices.Field("host_id"), hosts.Field("id")))
		q = q.Filter(sqlchemy.In(hosts.Field("zone_id"), subq))
	}
	if zone != nil {
		q = q.Join(hosts, sqlchemy.Equals(devices.Field("host_id"), hosts.Field("id")))
		q = q.Filter(sqlchemy.Equals(hosts.Field("zone_id"), zone.Id))
	}
	q = q.Distinct()

	rows, err := q.Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	gpus := make([]string, 0)
	for rows.Next() {
		var model string
		rows.Scan(&model)
		if len(model) > 0 {
			gpus = append(gpus, model)
		}
	}
	return gpus
}

func getNetworkCount(region *SCloudregion, zone *SZone) int {
	wires := WireManager.Query().SubQuery()
	networks := NetworkManager.Query().SubQuery()

	q := networks.Query()
	if region != nil {
		subq := getRegionZoneSubq(region)
		q = q.Join(wires, sqlchemy.Equals(networks.Field("wire_id"), wires.Field("id")))
		q = q.Filter(sqlchemy.In(wires.Field("zone_id"), subq))
	}
	if zone != nil {
		q = q.Join(wires, sqlchemy.Equals(networks.Field("wire_id"), wires.Field("id")))
		q = q.Filter(sqlchemy.Equals(wires.Field("zone_id"), zone.Id))
	}
	q = q.Filter(sqlchemy.Equals(networks.Field("status"), api.NETWORK_STATUS_AVAILABLE))

	return q.Count()
}

func isSchedPolicySupported(region *SCloudregion, zone *SZone) bool {
	return true
}

func getMinNicCount(region *SCloudregion, zone *SZone) int {
	if region != nil {
		return region.getMinNicCount()
	}
	if zone != nil {
		return zone.getMinNicCount()
	}
	return 0
}

func getMaxNicCount(region *SCloudregion, zone *SZone) int {
	if region != nil {
		return region.getMaxNicCount()
	}
	if zone != nil {
		return zone.getMaxNicCount()
	}
	return 0
}

func getMinDataDiskCount(region *SCloudregion, zone *SZone) int {
	if region != nil {
		return region.getMinDataDiskCount()
	}
	if zone != nil {
		return zone.getMinDataDiskCount()
	}
	return 0
}

func getMaxDataDiskCount(region *SCloudregion, zone *SZone) int {
	if region != nil {
		return region.getMaxDataDiskCount()
	}
	if zone != nil {
		return zone.getMaxDataDiskCount()
	}
	return 0
}

func isUsable(region *SCloudregion, zone *SZone) bool {
	if getNetworkCount(region, zone) > 0 {
		return true
	} else {
		return false
	}
}
