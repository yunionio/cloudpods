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
	"context"
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type DiskStats struct {
	HypervisorAvgIoLatencyUsecs                     string `json:"hypervisor_avg_io_latency_usecs"`
	HypervisorWriteIoBandwidthKBps                  string `json:"hypervisor_write_io_bandwidth_kBps"`
	ControllerRandomOpsPpm                          string `json:"controller.random_ops_ppm"`
	ControllerStorageTierSsdUsageBytes              string `json:"controller.storage_tier.ssd.usage_bytes"`
	ReadIoPpm                                       string `json:"read_io_ppm"`
	ControllerFrontendReadLatencyHistogram1000Us    string `json:"controller.frontend_read_latency_histogram_1000us"`
	ControllerNumIops                               string `json:"controller_num_iops"`
	ControllerFrontendWriteOps                      string `json:"controller.frontend_write_ops"`
	ControllerFrontendWriteLatencyHistogram10000Us  string `json:"controller.frontend_write_latency_histogram_10000us"`
	ControllerReadSizeHistogram1024KB               string `json:"controller.read_size_histogram_1024kB"`
	TotalReadIoTimeUsecs                            string `json:"total_read_io_time_usecs"`
	ControllerTotalReadIoTimeUsecs                  string `json:"controller_total_read_io_time_usecs"`
	ControllerWss3600SWriteMB                       string `json:"controller.wss_3600s_write_MB"`
	ControllerFrontendReadLatencyHistogram50000Us   string `json:"controller.frontend_read_latency_histogram_50000us"`
	ControllerFrontendReadLatencyHistogram2000Us    string `json:"controller.frontend_read_latency_histogram_2000us"`
	ControllerNumWriteIo                            string `json:"controller_num_write_io"`
	ControllerReadSourceCacheSsdBytes               string `json:"controller.read_source_cache_ssd_bytes"`
	ControllerReadSourceOplogBytes                  string `json:"controller.read_source_oplog_bytes"`
	ControllerReadSourceCacheDramBytes              string `json:"controller.read_source_cache_dram_bytes"`
	ControllerRandomReadOps                         string `json:"controller.random_read_ops"`
	ControllerTotalIoTimeUsecs                      string `json:"controller_total_io_time_usecs"`
	ControllerNumSeqIo                              string `json:"controller_num_seq_io"`
	ControllerTotalIoSizeKbytes                     string `json:"controller_total_io_size_kbytes"`
	ControllerWss120SWriteMB                        string `json:"controller.wss_120s_write_MB"`
	ControllerReadSourceBlockStoreBytes             string `json:"controller.read_source_block_store_bytes"`
	ControllerNumIo                                 string `json:"controller_num_io"`
	ControllerReadSourceEstoreZeroBytes             string `json:"controller.read_source_estore_zero_bytes"`
	ControllerNumRandomIo                           string `json:"controller_num_random_io"`
	HypervisorNumReadIo                             string `json:"hypervisor_num_read_io"`
	HypervisorTotalReadIoTimeUsecs                  string `json:"hypervisor_total_read_io_time_usecs"`
	NumIo                                           string `json:"num_io"`
	HypervisorNumWriteIo                            string `json:"hypervisor_num_write_io"`
	ControllerWriteSizeHistogram32KB                string `json:"controller.write_size_histogram_32kB"`
	ControllerFrontendReadLatencyHistogram20000Us   string `json:"controller.frontend_read_latency_histogram_20000us"`
	ControllerReadSizeHistogram32KB                 string `json:"controller.read_size_histogram_32kB"`
	HypervisorNumWriteIops                          string `json:"hypervisor_num_write_iops"`
	AvgIoLatencyUsecs                               string `json:"avg_io_latency_usecs"`
	ControllerWriteIoPpm                            string `json:"controller_write_io_ppm"`
	ControllerReadSourceEstoreSsdBytes              string `json:"controller.read_source_estore_ssd_bytes"`
	HypervisorTotalReadIoSizeKbytes                 string `json:"hypervisor_total_read_io_size_kbytes"`
	ControllerNumWriteIops                          string `json:"controller_num_write_iops"`
	TotalIoTimeUsecs                                string `json:"total_io_time_usecs"`
	ControllerWss3600SReadMB                        string `json:"controller.wss_3600s_read_MB"`
	ControllerSummaryReadSourceSsdBytesPerSec       string `json:"controller.summary_read_source_ssd_bytes_per_sec"`
	ControllerWriteSizeHistogram16KB                string `json:"controller.write_size_histogram_16kB"`
	TotalTransformedUsageBytes                      string `json:"total_transformed_usage_bytes"`
	AvgWriteIoLatencyUsecs                          string `json:"avg_write_io_latency_usecs"`
	ControllerCseTarget90PercentWriteMB             string `json:"controller.cse_target_90_percent_write_MB"`
	NumReadIo                                       string `json:"num_read_io"`
	HypervisorReadIoBandwidthKBps                   string `json:"hypervisor_read_io_bandwidth_kBps"`
	HypervisorTotalIoTimeUsecs                      string `json:"hypervisor_total_io_time_usecs"`
	NumRandomIo                                     string `json:"num_random_io"`
	ControllerWriteDestEstoreBytes                  string `json:"controller.write_dest_estore_bytes"`
	ControllerFrontendWriteLatencyHistogram5000Us   string `json:"controller.frontend_write_latency_histogram_5000us"`
	ControllerStorageTierDasSataPinnedUsageBytes    string `json:"controller.storage_tier.das-sata.pinned_usage_bytes"`
	NumWriteIo                                      string `json:"num_write_io"`
	ControllerFrontendWriteLatencyHistogram2000Us   string `json:"controller.frontend_write_latency_histogram_2000us"`
	ControllerRandomWriteOpsPerSec                  string `json:"controller.random_write_ops_per_sec"`
	ControllerFrontendWriteLatencyHistogram20000Us  string `json:"controller.frontend_write_latency_histogram_20000us"`
	IoBandwidthKBps                                 string `json:"io_bandwidth_kBps"`
	ControllerWriteSizeHistogram512KB               string `json:"controller.write_size_histogram_512kB"`
	ControllerReadSizeHistogram16KB                 string `json:"controller.read_size_histogram_16kB"`
	WriteIoPpm                                      string `json:"write_io_ppm"`
	ControllerAvgWriteIoLatencyUsecs                string `json:"controller_avg_write_io_latency_usecs"`
	ControllerFrontendReadLatencyHistogram100000Us  string `json:"controller.frontend_read_latency_histogram_100000us"`
	NumReadIops                                     string `json:"num_read_iops"`
	ControllerSummaryReadSourceHddBytesPerSec       string `json:"controller.summary_read_source_hdd_bytes_per_sec"`
	ControllerReadSourceExtentCacheBytes            string `json:"controller.read_source_extent_cache_bytes"`
	TimespanUsecs                                   string `json:"timespan_usecs"`
	ControllerNumReadIops                           string `json:"controller_num_read_iops"`
	ControllerFrontendReadLatencyHistogram10000Us   string `json:"controller.frontend_read_latency_histogram_10000us"`
	ControllerWriteSizeHistogram64KB                string `json:"controller.write_size_histogram_64kB"`
	ControllerFrontendWriteLatencyHistogram0Us      string `json:"controller.frontend_write_latency_histogram_0us"`
	ControllerFrontendWriteLatencyHistogram100000Us string `json:"controller.frontend_write_latency_histogram_100000us"`
	HypervisorNumIo                                 string `json:"hypervisor_num_io"`
	ControllerTotalTransformedUsageBytes            string `json:"controller_total_transformed_usage_bytes"`
	AvgReadIoLatencyUsecs                           string `json:"avg_read_io_latency_usecs"`
	ControllerTotalReadIoSizeKbytes                 string `json:"controller_total_read_io_size_kbytes"`
	ControllerReadIoPpm                             string `json:"controller_read_io_ppm"`
	ControllerFrontendOps                           string `json:"controller.frontend_ops"`
	ControllerWss120SReadMB                         string `json:"controller.wss_120s_read_MB"`
	ControllerReadSizeHistogram512KB                string `json:"controller.read_size_histogram_512kB"`
	HypervisorAvgReadIoLatencyUsecs                 string `json:"hypervisor_avg_read_io_latency_usecs"`
	ControllerWriteSizeHistogram1024KB              string `json:"controller.write_size_histogram_1024kB"`
	ControllerWriteDestBlockStoreBytes              string `json:"controller.write_dest_block_store_bytes"`
	ControllerReadSizeHistogram4KB                  string `json:"controller.read_size_histogram_4kB"`
	NumWriteIops                                    string `json:"num_write_iops"`
	ControllerRandomOpsPerSec                       string `json:"controller.random_ops_per_sec"`
	NumIops                                         string `json:"num_iops"`
	ControllerStorageTierCloudPinnedUsageBytes      string `json:"controller.storage_tier.cloud.pinned_usage_bytes"`
	ControllerAvgIoLatencyUsecs                     string `json:"controller_avg_io_latency_usecs"`
	ControllerReadSizeHistogram8KB                  string `json:"controller.read_size_histogram_8kB"`
	ControllerNumReadIo                             string `json:"controller_num_read_io"`
	ControllerSeqIoPpm                              string `json:"controller_seq_io_ppm"`
	ControllerReadIoBandwidthKBps                   string `json:"controller_read_io_bandwidth_kBps"`
	ControllerIoBandwidthKBps                       string `json:"controller_io_bandwidth_kBps"`
	ControllerReadSizeHistogram0KB                  string `json:"controller.read_size_histogram_0kB"`
	ControllerRandomOps                             string `json:"controller.random_ops"`
	HypervisorTimespanUsecs                         string `json:"hypervisor_timespan_usecs"`
	TotalReadIoSizeKbytes                           string `json:"total_read_io_size_kbytes"`
	HypervisorTotalIoSizeKbytes                     string `json:"hypervisor_total_io_size_kbytes"`
	ControllerFrontendOpsPerSec                     string `json:"controller.frontend_ops_per_sec"`
	ControllerWriteDestOplogBytes                   string `json:"controller.write_dest_oplog_bytes"`
	ControllerFrontendWriteLatencyHistogram1000Us   string `json:"controller.frontend_write_latency_histogram_1000us"`
	HypervisorNumReadIops                           string `json:"hypervisor_num_read_iops"`
	ControllerSummaryReadSourceCacheBytesPerSec     string `json:"controller.summary_read_source_cache_bytes_per_sec"`
	ControllerWriteIoBandwidthKBps                  string `json:"controller_write_io_bandwidth_kBps"`
	ControllerUserBytes                             string `json:"controller_user_bytes"`
	HypervisorAvgWriteIoLatencyUsecs                string `json:"hypervisor_avg_write_io_latency_usecs"`
	ControllerStorageTierSsdPinnedUsageBytes        string `json:"controller.storage_tier.ssd.pinned_usage_bytes"`
	ReadIoBandwidthKBps                             string `json:"read_io_bandwidth_kBps"`
	ControllerFrontendReadOps                       string `json:"controller.frontend_read_ops"`
	HypervisorNumIops                               string `json:"hypervisor_num_iops"`
	HypervisorIoBandwidthKBps                       string `json:"hypervisor_io_bandwidth_kBps"`
	ControllerWss120SUnionMB                        string `json:"controller.wss_120s_union_MB"`
	ControllerReadSourceEstoreHddBytes              string `json:"controller.read_source_estore_hdd_bytes"`
	ControllerRandomIoPpm                           string `json:"controller_random_io_ppm"`
	ControllerCseTarget90PercentReadMB              string `json:"controller.cse_target_90_percent_read_MB"`
	ControllerStorageTierDasSataUsageBytes          string `json:"controller.storage_tier.das-sata.usage_bytes"`
	ControllerFrontendReadLatencyHistogram5000Us    string `json:"controller.frontend_read_latency_histogram_5000us"`
	ControllerAvgReadIoSizeKbytes                   string `json:"controller_avg_read_io_size_kbytes"`
	WriteIoBandwidthKBps                            string `json:"write_io_bandwidth_kBps"`
	ControllerRandomReadOpsPerSec                   string `json:"controller.random_read_ops_per_sec"`
	ControllerReadSizeHistogram64KB                 string `json:"controller.read_size_histogram_64kB"`
	ControllerWss3600SUnionMB                       string `json:"controller.wss_3600s_union_MB"`
	RandomIoPpm                                     string `json:"random_io_ppm"`
	TotalUntransformedUsageBytes                    string `json:"total_untransformed_usage_bytes"`
	ControllerFrontendReadLatencyHistogram0Us       string `json:"controller.frontend_read_latency_histogram_0us"`
	ControllerRandomWriteOps                        string `json:"controller.random_write_ops"`
	ControllerAvgWriteIoSizeKbytes                  string `json:"controller_avg_write_io_size_kbytes"`
	ControllerAvgReadIoLatencyUsecs                 string `json:"controller_avg_read_io_latency_usecs"`
	TotalIoSizeKbytes                               string `json:"total_io_size_kbytes"`
	ControllerStorageTierCloudUsageBytes            string `json:"controller.storage_tier.cloud.usage_bytes"`
	ControllerFrontendWriteLatencyHistogram50000Us  string `json:"controller.frontend_write_latency_histogram_50000us"`
	ControllerWriteSizeHistogram8KB                 string `json:"controller.write_size_histogram_8kB"`
	ControllerTimespanUsecs                         string `json:"controller_timespan_usecs"`
	NumSeqIo                                        string `json:"num_seq_io"`
	ControllerWriteSizeHistogram4KB                 string `json:"controller.write_size_histogram_4kB"`
	SeqIoPpm                                        string `json:"seq_io_ppm"`
	ControllerWriteSizeHistogram0KB                 string `json:"controller.write_size_histogram_0kB"`
}

type SDisk struct {
	multicloud.STagBase
	multicloud.SDisk

	storage *SStorage

	VirtualDiskID         string    `json:"virtual_disk_id"`
	UUID                  string    `json:"uuid"`
	DeviceUUID            string    `json:"device_uuid"`
	NutanixNfsfilePath    string    `json:"nutanix_nfsfile_path"`
	DiskAddress           string    `json:"disk_address"`
	AttachedVMID          string    `json:"attached_vm_id"`
	AttachedVMUUID        string    `json:"attached_vm_uuid"`
	AttachedVmname        string    `json:"attached_vmname"`
	AttachedVolumeGroupID string    `json:"attached_volume_group_id"`
	DiskCapacityInBytes   int64     `json:"disk_capacity_in_bytes"`
	ClusterUUID           string    `json:"cluster_uuid"`
	StorageContainerID    string    `json:"storage_container_id"`
	StorageContainerUUID  string    `json:"storage_container_uuid"`
	FlashModeEnabled      string    `json:"flash_mode_enabled"`
	DataSourceURL         string    `json:"data_source_url"`
	Stats                 DiskStats `json:"stats"`

	isSys bool
}

func (self *SDisk) GetName() string {
	return self.DiskAddress
}

func (self *SDisk) GetId() string {
	return self.UUID
}

func (self *SDisk) GetGlobalId() string {
	return self.UUID
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDisk) GetAccessPath() string {
	return self.NutanixNfsfilePath
}

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	if info := strings.Split(self.DiskAddress, "."); len(info) > 0 {
		return info[0]
	}
	return "scsi"
}

func (self *SDisk) GetDiskType() string {
	if self.isSys {
		return api.DISK_TYPE_SYS
	}
	ins, err := self.storage.zone.region.GetInstance(self.AttachedVMUUID)
	if err != nil {
		return api.DISK_TYPE_DATA
	}
	for _, disk := range ins.VMDiskInfo {
		if disk.IsCdrom {
			continue
		}
		if disk.DiskAddress.VmdiskUUID == self.UUID {
			return api.DISK_TYPE_SYS
		} else {
			return api.DISK_TYPE_DATA
		}
	}
	return api.DISK_TYPE_DATA
}

func (self *SDisk) GetDiskFormat() string {
	return "raw"
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.DiskCapacityInBytes / 1024 / 1024)
}

func (self *SDisk) GetIsAutoDelete() bool {
	return true
}

func (self *SDisk) GetMountpoint() string {
	return ""
}

func (self *SDisk) GetStatus() string {
	return api.DISK_READY
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	ins, err := self.storage.zone.region.GetInstance(self.AttachedVMUUID)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%s)", self.AttachedVMUUID)
	}
	for _, disk := range ins.VMDiskInfo {
		if disk.DiskAddress.VmdiskUUID == self.UUID {
			params := map[string]interface{}{
				"vm_disks": []map[string]interface{}{
					{
						"disk_address":       disk.DiskAddress,
						"flash_mode_enabled": disk.FlashModeEnabled,
						"is_cdrom":           disk.IsCdrom,
						"is_empty":           disk.IsEmpty,
						"vm_disk_create": map[string]interface{}{
							"storage_container_uuid": disk.StorageContainerUUID,
							"size":                   sizeMb * 1024 * 1024,
						},
					},
				},
			}
			return self.storage.zone.region.update("vms", fmt.Sprintf("%s/disks/update", self.AttachedVMUUID), jsonutils.Marshal(params), nil)
		}
	}
	return cloudprovider.ErrNotSupported
}

func (self *SDisk) GetTemplateId() string {
	return ""
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetDisks(storageId, vmId string) ([]SDisk, error) {
	disks := []SDisk{}
	params := url.Values{}
	filter := []string{}
	if len(storageId) > 0 {
		filter = append(filter, "container_uuid=="+storageId)
	}
	if len(vmId) > 0 {
		filter = append(filter, "vm_uuid=="+vmId)
		filter = append(filter, "attach_vm_id=="+vmId)
	}
	if len(filter) > 0 {
		params.Set("filter_criteria", strings.Join(filter, ","))
	}
	return disks, self.listAll("virtual_disks", params, &disks)
}

func (self *SRegion) GetDisk(id string) (*SDisk, error) {
	disk := &SDisk{}
	return disk, self.get("virtual_disks", id, url.Values{}, disk)
}
