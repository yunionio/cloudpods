package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCapabilities struct {
	Hypervisors        []string `json:",allowempty"`
	StorageTypes       []string `json:",allowempty"`
	GPUModels          []string `json:",allowempty"`
	MinNicCount        int
	MaxNicCount        int
	MinDataDiskCount   int
	MaxDataDiskCount   int
	SchedPolicySupport bool
	Usable             bool
	Specs              jsonutils.JSONObject
}

func GetCapabilities(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, zone *SZone) (SCapabilities, error) {
	capa := SCapabilities{}
	capa.Hypervisors = getHypervisors(zone)
	capa.StorageTypes = getStorageTypes(zone)
	capa.GPUModels = getGPUs(zone)
	capa.SchedPolicySupport = isSchedPolicySupported(zone)
	capa.MinNicCount = getMinNicCount(zone)
	capa.MaxNicCount = getMaxNicCount(zone)
	capa.MinDataDiskCount = getMinDataDiskCount(zone)
	capa.MaxDataDiskCount = getMaxDataDiskCount(zone)
	capa.Usable = isUsable(zone)
	if query == nil {
		query = jsonutils.NewDict()
	}
	var err error
	if zone != nil {
		query.(*jsonutils.JSONDict).Add(jsonutils.NewString(zone.GetId()), "zone")
	}
	mans := []ISpecModelManager{HostManager, IsolatedDeviceManager}
	capa.Specs, err = GetModelsSpecs(ctx, userCred, query.(*jsonutils.JSONDict), mans...)
	return capa, err
}

func getHypervisors(zone *SZone) []string {
	q := HostManager.Query("host_type")
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
		rows.Scan(&hostType)
		if len(hostType) > 0 {
			hypervisors = append(hypervisors, HOSTTYPE_HYPERVISOR[hostType])
		}
	}
	return hypervisors
}

func getStorageTypes(zone *SZone) []string {
	q := StorageManager.Query("storage_type", "medium_type")
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	q = q.IsNotEmpty("storage_type").IsNotNull("storage_type")
	q = q.IsNotEmpty("medium_type").IsNotNull("medium_type")
	q = q.In("status", []string{STORAGE_ENABLED, STORAGE_ONLINE})
	q = q.IsTrue("enabled")
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

func getGPUs(zone *SZone) []string {
	devices := IsolatedDeviceManager.Query().SubQuery()
	hosts := HostManager.Query().SubQuery()

	q := devices.Query(devices.Field("model"))
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

func getNetworkCount(zone *SZone) int {
	networks := NetworkManager.Query().SubQuery()

	q := networks.Query()
	if zone != nil {
		wires := WireManager.Query().SubQuery()
		q = q.Join(wires, sqlchemy.Equals(networks.Field("wire_id"), wires.Field("id")))
		q = q.Filter(sqlchemy.Equals(wires.Field("zone_id"), zone.Id))
	}
	q = q.Filter(sqlchemy.Equals(networks.Field("status"), NETWORK_STATUS_AVAILABLE))

	return q.Count()
}

func isSchedPolicySupported(zone *SZone) bool {
	return true
}

func getMinNicCount(zone *SZone) int {
	if zone != nil {
		return zone.getMinNicCount()
	} else {
		return 0
	}
}

func getMaxNicCount(zone *SZone) int {
	if zone != nil {
		return zone.getMaxNicCount()
	} else {
		return 0
	}
}

func getMinDataDiskCount(zone *SZone) int {
	if zone != nil {
		return zone.getMinDataDiskCount()
	} else {
		return 0
	}
}

func getMaxDataDiskCount(zone *SZone) int {
	if zone != nil {
		return zone.getMaxDataDiskCount()
	} else {
		return 0
	}
}

func isUsable(zone *SZone) bool {
	if getNetworkCount(zone) > 0 {
		return true
	} else {
		return false
	}
}
