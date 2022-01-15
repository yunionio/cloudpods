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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type DownMigrateTimesInSecs struct {
	SSDSATA int `json:"SSD-SATA"`
	SSDPCIe int `json:"SSD-PCIe"`
	DASSATA int `json:"DAS-SATA"`
}

type MappedRemoteContainers struct {
}

type StorageStats struct {
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
	ControllerNumWriteIo                 string `json:"controller_num_write_io"`
	AvgReadIoLatencyUsecs                string `json:"avg_read_io_latency_usecs"`
	ControllerTotalIoTimeUsecs           string `json:"controller_total_io_time_usecs"`
	ControllerTotalReadIoSizeKbytes      string `json:"controller_total_read_io_size_kbytes"`
	ControllerNumSeqIo                   string `json:"controller_num_seq_io"`
	ControllerReadIoPpm                  string `json:"controller_read_io_ppm"`
	ControllerTotalIoSizeKbytes          string `json:"controller_total_io_size_kbytes"`
	ControllerNumIo                      string `json:"controller_num_io"`
	HypervisorAvgReadIoLatencyUsecs      string `json:"hypervisor_avg_read_io_latency_usecs"`
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
	HypervisorTimespanUsecs              string `json:"hypervisor_timespan_usecs"`
	HypervisorNumWriteIops               string `json:"hypervisor_num_write_iops"`
	TotalReadIoSizeKbytes                string `json:"total_read_io_size_kbytes"`
	HypervisorTotalIoSizeKbytes          string `json:"hypervisor_total_io_size_kbytes"`
	AvgIoLatencyUsecs                    string `json:"avg_io_latency_usecs"`
	HypervisorNumReadIops                string `json:"hypervisor_num_read_iops"`
	ControllerWriteIoBandwidthKBps       string `json:"controller_write_io_bandwidth_kBps"`
	ControllerWriteIoPpm                 string `json:"controller_write_io_ppm"`
	HypervisorAvgWriteIoLatencyUsecs     string `json:"hypervisor_avg_write_io_latency_usecs"`
	HypervisorTotalReadIoSizeKbytes      string `json:"hypervisor_total_read_io_size_kbytes"`
	ReadIoBandwidthKBps                  string `json:"read_io_bandwidth_kBps"`
	HypervisorNumIops                    string `json:"hypervisor_num_iops"`
	HypervisorIoBandwidthKBps            string `json:"hypervisor_io_bandwidth_kBps"`
	ControllerNumWriteIops               string `json:"controller_num_write_iops"`
	TotalIoTimeUsecs                     string `json:"total_io_time_usecs"`
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
	ControllerTimespanUsecs              string `json:"controller_timespan_usecs"`
	NumSeqIo                             string `json:"num_seq_io"`
	SeqIoPpm                             string `json:"seq_io_ppm"`
	WriteIoPpm                           string `json:"write_io_ppm"`
	ControllerAvgWriteIoLatencyUsecs     string `json:"controller_avg_write_io_latency_usecs"`
}
type StorageUsageStats struct {
	StorageUserUnreservedOwnUsageBytes               string `json:"storage.user_unreserved_own_usage_bytes"`
	StorageReservedFreeBytes                         string `json:"storage.reserved_free_bytes"`
	DataReductionOverallSavingRatioPpm               string `json:"data_reduction.overall.saving_ratio_ppm"`
	DataReductionUserSavedBytes                      string `json:"data_reduction.user_saved_bytes"`
	StorageTierDasSataUsageBytes                     string `json:"storage_tier.das-sata.usage_bytes"`
	DataReductionErasureCodingPostReductionBytes     string `json:"data_reduction.erasure_coding.post_reduction_bytes"`
	StorageReservedUsageBytes                        string `json:"storage.reserved_usage_bytes"`
	StorageUserUnreservedSharedUsageBytes            string `json:"storage.user_unreserved_shared_usage_bytes"`
	StorageUserUnreservedUsageBytes                  int64  `json:"storage.user_unreserved_usage_bytes"`
	StorageUsageBytes                                string `json:"storage.usage_bytes"`
	DataReductionCompressionUserSavedBytes           string `json:"data_reduction.compression.user_saved_bytes"`
	DataReductionErasureCodingUserPreReductionBytes  string `json:"data_reduction.erasure_coding.user_pre_reduction_bytes"`
	StorageUserUnreservedCapacityBytes               string `json:"storage.user_unreserved_capacity_bytes"`
	StorageUserCapacityBytes                         int64  `json:"storage.user_capacity_bytes"`
	StorageUserStoragePoolCapacityBytes              string `json:"storage.user_storage_pool_capacity_bytes"`
	DataReductionPreReductionBytes                   string `json:"data_reduction.pre_reduction_bytes"`
	DataReductionUserPreReductionBytes               string `json:"data_reduction.user_pre_reduction_bytes"`
	StorageUserOtherContainersReservedCapacityBytes  string `json:"storage.user_other_containers_reserved_capacity_bytes"`
	DataReductionErasureCodingPreReductionBytes      string `json:"data_reduction.erasure_coding.pre_reduction_bytes"`
	StorageCapacityBytes                             int64  `json:"storage.capacity_bytes"`
	StorageUserUnreservedFreeBytes                   string `json:"storage.user_unreserved_free_bytes"`
	DataReductionCloneUserSavedBytes                 string `json:"data_reduction.clone.user_saved_bytes"`
	DataReductionDedupPostReductionBytes             string `json:"data_reduction.dedup.post_reduction_bytes"`
	DataReductionCloneSavingRatioPpm                 string `json:"data_reduction.clone.saving_ratio_ppm"`
	StorageLogicalUsageBytes                         string `json:"storage.logical_usage_bytes"`
	DataReductionSavedBytes                          string `json:"data_reduction.saved_bytes"`
	StorageUserDiskPhysicalUsageBytes                string `json:"storage.user_disk_physical_usage_bytes"`
	StorageFreeBytes                                 string `json:"storage.free_bytes"`
	DataReductionCompressionPostReductionBytes       string `json:"data_reduction.compression.post_reduction_bytes"`
	DataReductionCompressionUserPostReductionBytes   string `json:"data_reduction.compression.user_post_reduction_bytes"`
	StorageUserFreeBytes                             string `json:"storage.user_free_bytes"`
	StorageUnreservedFreeBytes                       string `json:"storage.unreserved_free_bytes"`
	StorageUserContainerOwnUsageBytes                string `json:"storage.user_container_own_usage_bytes"`
	DataReductionCompressionSavingRatioPpm           string `json:"data_reduction.compression.saving_ratio_ppm"`
	StorageUserUsageBytes                            int64  `json:"storage.user_usage_bytes"`
	DataReductionErasureCodingUserSavedBytes         string `json:"data_reduction.erasure_coding.user_saved_bytes"`
	DataReductionDedupSavingRatioPpm                 string `json:"data_reduction.dedup.saving_ratio_ppm"`
	StorageUnreservedCapacityBytes                   string `json:"storage.unreserved_capacity_bytes"`
	StorageUserReservedUsageBytes                    string `json:"storage.user_reserved_usage_bytes"`
	DataReductionCompressionUserPreReductionBytes    string `json:"data_reduction.compression.user_pre_reduction_bytes"`
	DataReductionUserPostReductionBytes              string `json:"data_reduction.user_post_reduction_bytes"`
	DataReductionOverallUserSavedBytes               string `json:"data_reduction.overall.user_saved_bytes"`
	DataReductionErasureCodingParityBytes            string `json:"data_reduction.erasure_coding.parity_bytes"`
	DataReductionSavingRatioPpm                      string `json:"data_reduction.saving_ratio_ppm"`
	StorageUnreservedOwnUsageBytes                   string `json:"storage.unreserved_own_usage_bytes"`
	DataReductionErasureCodingSavingRatioPpm         string `json:"data_reduction.erasure_coding.saving_ratio_ppm"`
	StorageUserReservedCapacityBytes                 string `json:"storage.user_reserved_capacity_bytes"`
	DataReductionThinProvisionUserSavedBytes         string `json:"data_reduction.thin_provision.user_saved_bytes"`
	StorageDiskPhysicalUsageBytes                    string `json:"storage.disk_physical_usage_bytes"`
	DataReductionErasureCodingUserPostReductionBytes string `json:"data_reduction.erasure_coding.user_post_reduction_bytes"`
	DataReductionCompressionPreReductionBytes        string `json:"data_reduction.compression.pre_reduction_bytes"`
	DataReductionDedupPreReductionBytes              string `json:"data_reduction.dedup.pre_reduction_bytes"`
	DataReductionDedupUserSavedBytes                 string `json:"data_reduction.dedup.user_saved_bytes"`
	StorageUnreservedUsageBytes                      string `json:"storage.unreserved_usage_bytes"`
	StorageTierSsdUsageBytes                         string `json:"storage_tier.ssd.usage_bytes"`
	DataReductionPostReductionBytes                  string `json:"data_reduction.post_reduction_bytes"`
	DataReductionThinProvisionSavingRatioPpm         string `json:"data_reduction.thin_provision.saving_ratio_ppm"`
	StorageReservedCapacityBytes                     string `json:"storage.reserved_capacity_bytes"`
	StorageUserReservedFreeBytes                     string `json:"storage.user_reserved_free_bytes"`
}

type SStorage struct {
	multicloud.SStorageBase
	multicloud.STagBase

	zone *SZone

	StorageContainerUUID          string                 `json:"storage_container_uuid"`
	Name                          string                 `json:"name"`
	ClusterUUID                   string                 `json:"cluster_uuid"`
	MarkedForRemoval              bool                   `json:"marked_for_removal"`
	MaxCapacity                   int64                  `json:"max_capacity"`
	TotalExplicitReservedCapacity int                    `json:"total_explicit_reserved_capacity"`
	TotalImplicitReservedCapacity int                    `json:"total_implicit_reserved_capacity"`
	AdvertisedCapacity            interface{}            `json:"advertised_capacity"`
	ReplicationFactor             int                    `json:"replication_factor"`
	OplogReplicationFactor        int                    `json:"oplog_replication_factor"`
	NfsWhitelist                  []interface{}          `json:"nfs_whitelist"`
	NfsWhitelistInherited         bool                   `json:"nfs_whitelist_inherited"`
	RandomIoPreference            []string               `json:"random_io_preference"`
	SeqIoPreference               []string               `json:"seq_io_preference"`
	IlmPolicy                     interface{}            `json:"ilm_policy"`
	DownMigrateTimesInSecs        DownMigrateTimesInSecs `json:"down_migrate_times_in_secs"`
	ErasureCode                   string                 `json:"erasure_code"`
	InlineEcEnabled               interface{}            `json:"inline_ec_enabled"`
	PreferHigherEcfaultDomain     interface{}            `json:"prefer_higher_ecfault_domain"`
	ErasureCodeDelaySecs          interface{}            `json:"erasure_code_delay_secs"`
	FingerPrintOnWrite            string                 `json:"finger_print_on_write"`
	OnDiskDedup                   string                 `json:"on_disk_dedup"`
	CompressionEnabled            bool                   `json:"compression_enabled"`
	CompressionDelayInSecs        int                    `json:"compression_delay_in_secs"`
	IsNutanixManaged              interface{}            `json:"is_nutanix_managed"`
	EnableSoftwareEncryption      bool                   `json:"enable_software_encryption"`
	VstoreNameList                []string               `json:"vstore_name_list"`
	MappedRemoteContainers        MappedRemoteContainers `json:"mapped_remote_containers"`
	Stats                         StorageStats           `json:"stats"`
	UsageStats                    StorageUsageStats      `json:"usage_stats"`
	Encrypted                     interface{}            `json:"encrypted"`
}

func (self *SStorage) GetName() string {
	return self.Name
}

func (self *SStorage) GetId() string {
	return self.StorageContainerUUID
}

func (self *SStorage) GetGlobalId() string {
	return self.StorageContainerUUID
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks(self.GetGlobalId(), "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisks")
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		disks[i].storage = self
		ret = append(ret, &disks[i])
	}
	return ret, nil
}

func (self *SStorage) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStorage) GetCapacityMB() int64 {
	return self.UsageStats.StorageUserCapacityBytes / 1024 / 1024
}

func (self *SStorage) GetCapacityUsedMB() int64 {
	return self.UsageStats.StorageUserUsageBytes / 1024 / 1024
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	if disk.StorageContainerUUID != self.GetGlobalId() {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}
	disk.storage = self
	return disk, nil
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return &SStoragecache{storage: self, region: self.zone.region}
}

func (self *SRegion) GetStorages() ([]SStorage, error) {
	storages := []SStorage{}
	err := self.listAll("storage_containers", nil, &storages)
	if err != nil {
		return nil, err
	}
	ret := []SStorage{}
	for i := range storages {
		if storages[i].Name == "NutanixManagementShare" { // https://portal.nutanix.com/page/documents/details?targetId=Web-Console-Guide-Prism-v5_15-ZH:wc-container-create-wc-t.html
			continue
		}
		ret = append(ret, storages[i])
	}
	return ret, nil
}

func (self *SRegion) GetStorage(id string) (*SStorage, error) {
	storage := &SStorage{}
	return storage, self.get("storage_containers", id, nil, storage)
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetMediumType() string {
	return api.DISK_TYPE_SSD
}

func (self *SStorage) GetMountPoint() string {
	return ""
}

func (self *SStorage) GetStatus() string {
	return api.STORAGE_ONLINE
}

func (self *SStorage) Refresh() error {
	storage, err := self.zone.region.GetStorage(self.GetGlobalId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, storage)
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SStorage) GetStorageType() string {
	return api.STORAGE_LOCAL
}

func (self *SStorage) IsSysDiskStore() bool {
	return true
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	zones, err := self.GetIZones()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIZones")
	}
	for i := range zones {
		storage, err := zones[i].GetIStorageById(id)
		if err == nil && storage != nil {
			return storage, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}
