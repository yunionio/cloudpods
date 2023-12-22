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

package nutanix

import (
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.STagBase
	multicloud.SHostBase
	firstHost bool

	zone *SZone

	ServiceVmid                            string                                 `json:"service_vmid"`
	UUID                                   string                                 `json:"uuid"`
	DiskHardwareConfigs                    DiskHardwareConfigs                    `json:"disk_hardware_configs"`
	Name                                   string                                 `json:"name"`
	ServiceVmexternalIP                    string                                 `json:"service_vmexternal_ip"`
	ServiceVmnatIP                         string                                 `json:"service_vmnat_ip"`
	ServiceVmnatPort                       string                                 `json:"service_vmnat_port"`
	OplogDiskPct                           float64                                `json:"oplog_disk_pct"`
	OplogDiskSize                          int64                                  `json:"oplog_disk_size"`
	HypervisorKey                          string                                 `json:"hypervisor_key"`
	HypervisorAddress                      string                                 `json:"hypervisor_address"`
	HypervisorUsername                     string                                 `json:"hypervisor_username"`
	HypervisorPassword                     string                                 `json:"hypervisor_password"`
	BackplaneIP                            string                                 `json:"backplane_ip"`
	ControllerVMBackplaneIP                string                                 `json:"controller_vm_backplane_ip"`
	RdmaBackplaneIps                       string                                 `json:"rdma_backplane_ips"`
	ManagementServerName                   string                                 `json:"management_server_name"`
	IpmiAddress                            string                                 `json:"ipmi_address"`
	IpmiUsername                           string                                 `json:"ipmi_username"`
	IpmiPassword                           string                                 `json:"ipmi_password"`
	Monitored                              bool                                   `json:"monitored"`
	Position                               Position                               `json:"position"`
	Serial                                 string                                 `json:"serial"`
	BlockSerial                            string                                 `json:"block_serial"`
	BlockModel                             string                                 `json:"block_model"`
	BlockModelName                         string                                 `json:"block_model_name"`
	BlockLocation                          string                                 `json:"block_location"`
	HostMaintenanceModeReason              string                                 `json:"host_maintenance_mode_reason"`
	HypervisorState                        string                                 `json:"hypervisor_state"`
	AcropolisConnectionState               string                                 `json:"acropolis_connection_state"`
	MetadataStoreStatus                    string                                 `json:"metadata_store_status"`
	MetadataStoreStatusMessage             string                                 `json:"metadata_store_status_message"`
	State                                  string                                 `json:"state"`
	DynamicRingChangingNode                string                                 `json:"dynamic_ring_changing_node"`
	RemovalStatus                          []string                               `json:"removal_status"`
	VzoneName                              string                                 `json:"vzone_name"`
	CPUModel                               string                                 `json:"cpu_model"`
	NumCPUCores                            int                                    `json:"num_cpu_cores"`
	NumCPUThreads                          int                                    `json:"num_cpu_threads"`
	NumCPUSockets                          int                                    `json:"num_cpu_sockets"`
	CPUFrequencyInHz                       int64                                  `json:"cpu_frequency_in_hz"`
	CPUCapacityInHz                        int64                                  `json:"cpu_capacity_in_hz"`
	MemoryCapacityInBytes                  int64                                  `json:"memory_capacity_in_bytes"`
	HypervisorFullName                     string                                 `json:"hypervisor_full_name"`
	HypervisorType                         string                                 `json:"hypervisor_type"`
	NumVms                                 int                                    `json:"num_vms"`
	BootTimeInUsecs                        int64                                  `json:"boot_time_in_usecs"`
	IsDegraded                             bool                                   `json:"is_degraded"`
	IsSecureBooted                         bool                                   `json:"is_secure_booted"`
	IsHardwareVirtualized                  bool                                   `json:"is_hardware_virtualized"`
	FailoverClusterFqdn                    string                                 `json:"failover_cluster_fqdn"`
	FailoverClusterNodeState               string                                 `json:"failover_cluster_node_state"`
	RebootPending                          bool                                   `json:"reboot_pending"`
	DefaultVMLocation                      string                                 `json:"default_vm_location"`
	DefaultVMStorageContainerID            string                                 `json:"default_vm_storage_container_id"`
	DefaultVMStorageContainerUUID          string                                 `json:"default_vm_storage_container_uuid"`
	DefaultVhdLocation                     string                                 `json:"default_vhd_location"`
	DefaultVhdStorageContainerID           string                                 `json:"default_vhd_storage_container_id"`
	DefaultVhdStorageContainerUUID         string                                 `json:"default_vhd_storage_container_uuid"`
	BiosVersion                            string                                 `json:"bios_version"`
	BiosModel                              string                                 `json:"bios_model"`
	BmcVersion                             string                                 `json:"bmc_version"`
	BmcModel                               string                                 `json:"bmc_model"`
	HbaFirmwaresList                       string                                 `json:"hba_firmwares_list"`
	ClusterUUID                            string                                 `json:"cluster_uuid"`
	Stats                                  Stats                                  `json:"stats"`
	UsageStats                             UsageStats                             `json:"usage_stats"`
	HasCsr                                 bool                                   `json:"has_csr"`
	HostNicIds                             []string                               `json:"host_nic_ids"`
	HostGpus                               string                                 `json:"host_gpus"`
	GpuDriverVersion                       string                                 `json:"gpu_driver_version"`
	HostType                               string                                 `json:"host_type"`
	KeyManagementDeviceToCertificateStatus KeyManagementDeviceToCertificateStatus `json:"key_management_device_to_certificate_status"`
	HostInMaintenanceMode                  string                                 `json:"host_in_maintenance_mode"`
}

type Num1 struct {
	SerialNumber           string `json:"serial_number"`
	DiskID                 string `json:"disk_id"`
	DiskUUID               string `json:"disk_uuid"`
	Location               int    `json:"location"`
	Bad                    bool   `json:"bad"`
	Mounted                bool   `json:"mounted"`
	MountPath              string `json:"mount_path"`
	Model                  string `json:"model"`
	Vendor                 string `json:"vendor"`
	BootDisk               bool   `json:"boot_disk"`
	OnlyBootDisk           bool   `json:"only_boot_disk"`
	UnderDiagnosis         bool   `json:"under_diagnosis"`
	BackgroundOperation    string `json:"background_operation"`
	CurrentFirmwareVersion string `json:"current_firmware_version"`
	TargetFirmwareVersion  string `json:"target_firmware_version"`
	CanAddAsNewDisk        bool   `json:"can_add_as_new_disk"`
	CanAddAsOldDisk        bool   `json:"can_add_as_old_disk"`
}
type Num2 struct {
	SerialNumber           string `json:"serial_number"`
	DiskID                 string `json:"disk_id"`
	DiskUUID               string `json:"disk_uuid"`
	Location               int    `json:"location"`
	Bad                    bool   `json:"bad"`
	Mounted                bool   `json:"mounted"`
	MountPath              string `json:"mount_path"`
	Model                  string `json:"model"`
	Vendor                 string `json:"vendor"`
	BootDisk               bool   `json:"boot_disk"`
	OnlyBootDisk           bool   `json:"only_boot_disk"`
	UnderDiagnosis         bool   `json:"under_diagnosis"`
	BackgroundOperation    string `json:"background_operation"`
	CurrentFirmwareVersion string `json:"current_firmware_version"`
	TargetFirmwareVersion  string `json:"target_firmware_version"`
	CanAddAsNewDisk        bool   `json:"can_add_as_new_disk"`
	CanAddAsOldDisk        bool   `json:"can_add_as_old_disk"`
}
type DiskHardwareConfigs struct {
	Num1 Num1 `json:"1"`
	Num2 Num2 `json:"2"`
}
type Position struct {
	Ordinal          int    `json:"ordinal"`
	Name             string `json:"name"`
	PhysicalPosition string `json:"physical_position"`
}
type Stats struct {
	HypervisorAvgIoLatencyUsecs          string `json:"hypervisor_avg_io_latency_usecs"`
	NumReadIops                          string `json:"num_read_iops"`
	HypervisorWriteIoBandwidthKBps       string `json:"hypervisor_write_io_bandwidth_kBps"`
	TimespanUsecs                        string `json:"timespan_usecs"`
	ControllerNumReadIops                string `json:"controller_num_read_iops"`
	ReadIoPpm                            string `json:"read_io_ppm"`
	ControllerNumIops                    string `json:"controller_num_iops"`
	TotalReadIoTimeUsecs                 string `json:"total_read_io_time_usecs"`
	ControllerTotalReadIoTimeUsecs       string `json:"controller_total_read_io_time_usecs"`
	HypervisorNumIo                      string `json:"hypervisor_num_io"`
	ControllerTotalTransformedUsageBytes string `json:"controller_total_transformed_usage_bytes"`
	HypervisorCPUUsagePpm                string `json:"hypervisor_cpu_usage_ppm"`
	ControllerNumWriteIo                 string `json:"controller_num_write_io"`
	AvgReadIoLatencyUsecs                string `json:"avg_read_io_latency_usecs"`
	ContentCacheLogicalSsdUsageBytes     string `json:"content_cache_logical_ssd_usage_bytes"`
	ControllerTotalIoTimeUsecs           string `json:"controller_total_io_time_usecs"`
	ControllerTotalReadIoSizeKbytes      string `json:"controller_total_read_io_size_kbytes"`
	ControllerNumSeqIo                   string `json:"controller_num_seq_io"`
	ControllerReadIoPpm                  string `json:"controller_read_io_ppm"`
	ContentCacheNumLookups               string `json:"content_cache_num_lookups"`
	ControllerTotalIoSizeKbytes          string `json:"controller_total_io_size_kbytes"`
	ContentCacheHitPpm                   string `json:"content_cache_hit_ppm"`
	ControllerNumIo                      string `json:"controller_num_io"`
	HypervisorAvgReadIoLatencyUsecs      string `json:"hypervisor_avg_read_io_latency_usecs"`
	ContentCacheNumDedupRefCountPph      string `json:"content_cache_num_dedup_ref_count_pph"`
	NumWriteIops                         string `json:"num_write_iops"`
	ControllerNumRandomIo                string `json:"controller_num_random_io"`
	NumIops                              string `json:"num_iops"`
	HypervisorNumReadIo                  string `json:"hypervisor_num_read_io"`
	HypervisorTotalReadIoTimeUsecs       string `json:"hypervisor_total_read_io_time_usecs"`
	ControllerAvgIoLatencyUsecs          string `json:"controller_avg_io_latency_usecs"`
	NumIo                                string `json:"num_io"`
	ControllerNumReadIo                  string `json:"controller_num_read_io"`
	HypervisorNumWriteIo                 string `json:"hypervisor_num_write_io"`
	ControllerSeqIoPpm                   string `json:"controller_seq_io_ppm"`
	ControllerReadIoBandwidthKBps        string `json:"controller_read_io_bandwidth_kBps"`
	ControllerIoBandwidthKBps            string `json:"controller_io_bandwidth_kBps"`
	HypervisorNumReceivedBytes           string `json:"hypervisor_num_received_bytes"`
	HypervisorTimespanUsecs              string `json:"hypervisor_timespan_usecs"`
	HypervisorNumWriteIops               string `json:"hypervisor_num_write_iops"`
	TotalReadIoSizeKbytes                string `json:"total_read_io_size_kbytes"`
	HypervisorTotalIoSizeKbytes          string `json:"hypervisor_total_io_size_kbytes"`
	AvgIoLatencyUsecs                    string `json:"avg_io_latency_usecs"`
	HypervisorNumReadIops                string `json:"hypervisor_num_read_iops"`
	ContentCacheSavedSsdUsageBytes       string `json:"content_cache_saved_ssd_usage_bytes"`
	ControllerWriteIoBandwidthKBps       string `json:"controller_write_io_bandwidth_kBps"`
	ControllerWriteIoPpm                 string `json:"controller_write_io_ppm"`
	HypervisorAvgWriteIoLatencyUsecs     string `json:"hypervisor_avg_write_io_latency_usecs"`
	HypervisorNumTransmittedBytes        string `json:"hypervisor_num_transmitted_bytes"`
	HypervisorTotalReadIoSizeKbytes      string `json:"hypervisor_total_read_io_size_kbytes"`
	ReadIoBandwidthKBps                  string `json:"read_io_bandwidth_kBps"`
	HypervisorMemoryUsagePpm             string `json:"hypervisor_memory_usage_ppm"`
	HypervisorNumIops                    string `json:"hypervisor_num_iops"`
	HypervisorIoBandwidthKBps            string `json:"hypervisor_io_bandwidth_kBps"`
	ControllerNumWriteIops               string `json:"controller_num_write_iops"`
	TotalIoTimeUsecs                     string `json:"total_io_time_usecs"`
	ContentCachePhysicalSsdUsageBytes    string `json:"content_cache_physical_ssd_usage_bytes"`
	ControllerRandomIoPpm                string `json:"controller_random_io_ppm"`
	ControllerAvgReadIoSizeKbytes        string `json:"controller_avg_read_io_size_kbytes"`
	TotalTransformedUsageBytes           string `json:"total_transformed_usage_bytes"`
	AvgWriteIoLatencyUsecs               string `json:"avg_write_io_latency_usecs"`
	NumReadIo                            string `json:"num_read_io"`
	WriteIoBandwidthKBps                 string `json:"write_io_bandwidth_kBps"`
	HypervisorReadIoBandwidthKBps        string `json:"hypervisor_read_io_bandwidth_kBps"`
	RandomIoPpm                          string `json:"random_io_ppm"`
	TotalUntransformedUsageBytes         string `json:"total_untransformed_usage_bytes"`
	HypervisorTotalIoTimeUsecs           string `json:"hypervisor_total_io_time_usecs"`
	NumRandomIo                          string `json:"num_random_io"`
	ControllerAvgWriteIoSizeKbytes       string `json:"controller_avg_write_io_size_kbytes"`
	ControllerAvgReadIoLatencyUsecs      string `json:"controller_avg_read_io_latency_usecs"`
	NumWriteIo                           string `json:"num_write_io"`
	TotalIoSizeKbytes                    string `json:"total_io_size_kbytes"`
	IoBandwidthKBps                      string `json:"io_bandwidth_kBps"`
	ContentCachePhysicalMemoryUsageBytes string `json:"content_cache_physical_memory_usage_bytes"`
	ControllerTimespanUsecs              string `json:"controller_timespan_usecs"`
	NumSeqIo                             string `json:"num_seq_io"`
	ContentCacheSavedMemoryUsageBytes    string `json:"content_cache_saved_memory_usage_bytes"`
	SeqIoPpm                             string `json:"seq_io_ppm"`
	WriteIoPpm                           string `json:"write_io_ppm"`
	ControllerAvgWriteIoLatencyUsecs     string `json:"controller_avg_write_io_latency_usecs"`
	ContentCacheLogicalMemoryUsageBytes  string `json:"content_cache_logical_memory_usage_bytes"`
}
type UsageStats struct {
	StorageTierDasSataUsageBytes    string `json:"storage_tier.das-sata.usage_bytes"`
	StorageCapacityBytes            string `json:"storage.capacity_bytes"`
	StorageLogicalUsageBytes        string `json:"storage.logical_usage_bytes"`
	StorageTierDasSataCapacityBytes string `json:"storage_tier.das-sata.capacity_bytes"`
	StorageFreeBytes                string `json:"storage.free_bytes"`
	StorageTierSsdUsageBytes        string `json:"storage_tier.ssd.usage_bytes"`
	StorageTierSsdCapacityBytes     string `json:"storage_tier.ssd.capacity_bytes"`
	StorageTierDasSataFreeBytes     string `json:"storage_tier.das-sata.free_bytes"`
	StorageUsageBytes               string `json:"storage.usage_bytes"`
	StorageTierSsdFreeBytes         string `json:"storage_tier.ssd.free_bytes"`
}
type KeyManagementDeviceToCertificateStatus struct {
}

func (self *SRegion) GetHosts() ([]SHost, error) {
	hosts := []SHost{}
	return hosts, self.listAll("hosts", nil, &hosts)
}

func (self *SRegion) GetHost(id string) (*SHost, error) {
	host := &SHost{}
	return host, self.cli.get("hosts", id, nil, host)
}

func (self *SHost) GetName() string {
	return self.Name
}

func (self *SHost) GetId() string {
	return self.UUID
}

func (self *SHost) GetGlobalId() string {
	return self.UUID
}

func (self *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	image, err := self.zone.region.GetImage(opts.ExternalImageId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage")
	}
	disks := []map[string]interface{}{
		{
			"disk_address": map[string]interface{}{
				"device_bus":   "ide",
				"device_index": 0,
			},
			"is_cdrom": true,
			"is_empty": true,
		},
		{
			"disk_address": map[string]interface{}{
				"device_bus":   "scsi",
				"device_index": 0,
			},
			"is_cdrom": false,
			"vm_disk_clone": map[string]interface{}{
				"disk_address": map[string]string{
					"vmdisk_uuid": image.VMDiskID,
				},
				"minimum_size": opts.SysDisk.SizeGB * 1024 * 1024 * 1024,
			},
		},
	}
	for i, disk := range opts.DataDisks {
		disks = append(disks, map[string]interface{}{
			"disk_address": map[string]interface{}{
				"device_bus":   "scsi",
				"device_index": i + 1,
			},
			"is_cdrom": false,
			"vm_disk_create": map[string]interface{}{
				"size":                   disk.SizeGB * 1024 * 1024 * 1024,
				"storage_container_uuid": disk.StorageExternalId,
			},
		})
	}
	nic := map[string]interface{}{
		"network_uuid": opts.ExternalVpcId,
	}
	if len(opts.IpAddr) > 0 {
		nic["requested_ip_address"] = opts.IpAddr
	}
	params := map[string]interface{}{
		"boot": map[string]interface{}{
			"boot_device_order": []string{"CDROM", "DISK", "NIC"},
			"uefi_boot":         false,
		},
		"description":        opts.Description,
		"hypervisor_type":    "ACROPOLIS",
		"memory_mb":          opts.MemoryMB,
		"name":               opts.Name,
		"num_cores_per_vcpu": 1,
		"num_vcpus":          opts.Cpu,
		"timezone":           "UTC",
		"vm_customization_config": map[string]interface{}{
			"files_to_inject_list": []string{},
			"userdata":             opts.UserData,
		},
		"vm_disks": disks,
		"vm_features": map[string]interface{}{
			"AGENT_VM": false,
		},
		"vm_nics": []map[string]interface{}{
			nic,
		},
	}
	ret := struct {
		TaskUUID string
	}{}
	err = self.zone.region.post("vms", jsonutils.Marshal(params), &ret)
	if err != nil {
		return nil, err
	}
	resId, err := self.zone.region.cli.wait(ret.TaskUUID)
	if err != nil {
		return nil, err
	}
	vm, err := self.zone.region.GetInstance(resId)
	if err != nil {
		return nil, err
	}
	vm.host = self
	return vm, nil
}

func (self *SHost) GetAccessIp() string {
	return self.HypervisorAddress
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetCpuCmtbound() float32 {
	return 16.0
}

func (self *SHost) GetMemCmtbound() float32 {
	return 1.5
}

func (self *SHost) GetCpuCount() int {
	return self.NumCPUCores * self.NumCPUSockets
}

func (self *SHost) GetNodeCount() int8 {
	return int8(self.NumCPUSockets)
}

func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetCpuDesc() string {
	return self.CPUModel
}

func (self *SHost) GetCpuMhz() int {
	return int(self.CPUCapacityInHz / 1000 / 1000)
}

func (self *SHost) GetMemSizeMB() int {
	return int(self.MemoryCapacityInBytes / 1024 / 1024)
}

func (self *SHost) GetStorageSizeMB() int64 {
	sizeBytes, _ := strconv.Atoi(self.UsageStats.StorageCapacityBytes)
	return int64(sizeBytes) / 1024 / 1024
}

func (self *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_NUTANIX
}

func (self *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.getIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return ""
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) GetSN() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_NUTANIX), "manufacture")
	return info
}

func (self *SHost) IsEmulated() bool {
	return false
}

func (self *SHost) Refresh() error {
	host, err := self.zone.region.GetHost(self.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, host)
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorages()
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorageById(id)
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances()
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstances")
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		if vms[i].HostUUID == self.UUID || (self.firstHost && len(vms[i].HostUUID) == 0) {
			vms[i].host = self
			ret = append(ret, &vms[i])
		}
	}
	return ret, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.zone.region.GetInstance(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstance")
	}
	if len(vm.HostUUID) > 0 && vm.HostUUID != self.UUID {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "vm not locate host %s, it locate host %s", self.Name, vm.HostUUID)
	}
	vm.host = self
	return vm, nil
}

func (self *SHost) getIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := self.zone.region.GetIVpcs()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		wires, err := vpcs[i].GetIWires()
		if err != nil {
			return nil, errors.Wrapf(err, "GetIWires")
		}
		ret = append(ret, wires...)
	}
	return ret, nil
}
