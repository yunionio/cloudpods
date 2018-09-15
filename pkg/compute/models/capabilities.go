package models

import (
	"fmt"

	"yunion.io/x/sqlchemy"
)


type SCapabilities struct {
	Hypervisors        []string
	StorageTypes       []string
	GPUModels          []string
	MinNicCount        int
	MaxNicCount        int
	MinDataDiskCount   int
	MaxDataDiskCount   int
	SchedPolicySupport bool
	Usable             bool
}

func GetCapabilities(zone *SZone) SCapabilities {
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
	return capa
}

func getHypervisors(zone *SZone) []string {
	q := HostManager.Query("host_type")
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	q = q.IsNotEmpty("host_type").IsNotNull("host_type")
	q = q.Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil
	}
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
	q = q.Distinct()
	rows, err := q.Rows()
	if err != nil {
		return nil
	}
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
	if zone != nil {
		return !zone.isManaged()
	} else {
		return true
	}
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