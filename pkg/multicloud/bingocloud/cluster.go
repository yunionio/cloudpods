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

package bingocloud

type RackableUnits struct {
	Id               int    `json:"id"`
	RackableUnitUUID string `json:"rackable_unit_uuid"`
	Model            string `json:"model"`
	ModelName        string `json:"model_name"`
	//Location         interface{} `json:"location"`
	Serial    string   `json:"serial"`
	Positions []string `json:"positions"`
	Nodes     []int    `json:"nodes"`
	NodeUuids []string `json:"node_uuids"`
}

type ClusterRedundancyState struct {
	CurrentRedundancyFactor int              `json:"current_redundancy_factor"`
	DesiredRedundancyFactor int              `json:"desired_redundancy_factor"`
	RedundancyStatus        RedundancyStatus `json:"redundancy_status"`
}

type RedundancyStatus struct {
	KCassandraPrepareDone bool `json:"kCassandraPrepareDone"`
	KZookeeperPrepareDone bool `json:"kZookeeperPrepareDone"`
}

type SecurityComplianceConfig struct {
	Schedule                   string `json:"schedule"`
	EnableAide                 bool   `json:"enable_aide"`
	EnableCore                 bool   `json:"enable_core"`
	EnableHighStrengthPassword bool   `json:"enable_high_strength_password"`
	EnableBanner               bool   `json:"enable_banner"`
	EnableSnmpv3Only           bool   `json:"enable_snmpv3_only"`
}
type HypervisorSecurityComplianceConfig struct {
	Schedule                   string `json:"schedule"`
	EnableAide                 bool   `json:"enable_aide"`
	EnableCore                 bool   `json:"enable_core"`
	EnableHighStrengthPassword bool   `json:"enable_high_strength_password"`
	EnableBanner               bool   `json:"enable_banner"`
}

type HypervisorLldpConfig struct {
	EnableLldpTx bool `json:"enable_lldp_tx"`
}

type ClusterStats struct {
	HypervisorAvgIoLatencyUsecs          string `json:"hypervisor_avg_io_latency_usecs"`
	NumReadIops                          string `json:"num_read_iops"`
	HypervisorWriteIoBandwidthKBps       string `json:"hypervisor_write_io_bandwidth_kBps"`
	TimespanUsecs                        string `json:"timespan_usecs"`
	ControllerNumReadIops                string `json:"controller_num_read_iops"`
	ReadIoPpm                            string `json:"read_io_ppm"`
	ControllerNumIops                    string `json:"controller_num_iops"`
	TotalReadIoTimeUsecs                 string `json:"total_read_io_time_usecs"`
	ControllerTotalReadIoTimeUsecs       string `json:"controller_total_read_io_time_usecs"`
	ReplicationTransmittedBandwidthKBps  string `json:"replication_transmitted_bandwidth_kBps"`
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
	ReplicationReceivedBandwidthKBps     string `json:"replication_received_bandwidth_kBps"`
	HypervisorNumReadIo                  string `json:"hypervisor_num_read_io"`
	HypervisorTotalReadIoTimeUsecs       string `json:"hypervisor_total_read_io_time_usecs"`
	ControllerAvgIoLatencyUsecs          string `json:"controller_avg_io_latency_usecs"`
	HypervisorHypervCPUUsagePpm          string `json:"hypervisor_hyperv_cpu_usage_ppm"`
	NumIo                                string `json:"num_io"`
	ControllerNumReadIo                  string `json:"controller_num_read_io"`
	HypervisorNumWriteIo                 string `json:"hypervisor_num_write_io"`
	ControllerSeqIoPpm                   string `json:"controller_seq_io_ppm"`
	ControllerReadIoBandwidthKBps        string `json:"controller_read_io_bandwidth_kBps"`
	ControllerIoBandwidthKBps            string `json:"controller_io_bandwidth_kBps"`
	HypervisorHypervMemoryUsagePpm       string `json:"hypervisor_hyperv_memory_usage_ppm"`
	HypervisorTimespanUsecs              string `json:"hypervisor_timespan_usecs"`
	HypervisorNumWriteIops               string `json:"hypervisor_num_write_iops"`
	ReplicationNumTransmittedBytes       string `json:"replication_num_transmitted_bytes"`
	TotalReadIoSizeKbytes                string `json:"total_read_io_size_kbytes"`
	HypervisorTotalIoSizeKbytes          string `json:"hypervisor_total_io_size_kbytes"`
	AvgIoLatencyUsecs                    string `json:"avg_io_latency_usecs"`
	HypervisorNumReadIops                string `json:"hypervisor_num_read_iops"`
	ContentCacheSavedSsdUsageBytes       string `json:"content_cache_saved_ssd_usage_bytes"`
	ControllerWriteIoBandwidthKBps       string `json:"controller_write_io_bandwidth_kBps"`
	ControllerWriteIoPpm                 string `json:"controller_write_io_ppm"`
	HypervisorAvgWriteIoLatencyUsecs     string `json:"hypervisor_avg_write_io_latency_usecs"`
	HypervisorTotalReadIoSizeKbytes      string `json:"hypervisor_total_read_io_size_kbytes"`
	ReadIoBandwidthKBps                  string `json:"read_io_bandwidth_kBps"`
	HypervisorEsxMemoryUsagePpm          string `json:"hypervisor_esx_memory_usage_ppm"`
	HypervisorMemoryUsagePpm             string `json:"hypervisor_memory_usage_ppm"`
	HypervisorNumIops                    string `json:"hypervisor_num_iops"`
	HypervisorIoBandwidthKBps            string `json:"hypervisor_io_bandwidth_kBps"`
	ControllerNumWriteIops               string `json:"controller_num_write_iops"`
	TotalIoTimeUsecs                     string `json:"total_io_time_usecs"`
	HypervisorKvmCPUUsagePpm             string `json:"hypervisor_kvm_cpu_usage_ppm"`
	ContentCachePhysicalSsdUsageBytes    string `json:"content_cache_physical_ssd_usage_bytes"`
	ControllerRandomIoPpm                string `json:"controller_random_io_ppm"`
	ControllerAvgReadIoSizeKbytes        string `json:"controller_avg_read_io_size_kbytes"`
	TotalTransformedUsageBytes           string `json:"total_transformed_usage_bytes"`
	AvgWriteIoLatencyUsecs               string `json:"avg_write_io_latency_usecs"`
	NumReadIo                            string `json:"num_read_io"`
	WriteIoBandwidthKBps                 string `json:"write_io_bandwidth_kBps"`
	HypervisorReadIoBandwidthKBps        string `json:"hypervisor_read_io_bandwidth_kBps"`
	RandomIoPpm                          string `json:"random_io_ppm"`
	ContentCacheNumHits                  string `json:"content_cache_num_hits"`
	TotalUntransformedUsageBytes         string `json:"total_untransformed_usage_bytes"`
	HypervisorTotalIoTimeUsecs           string `json:"hypervisor_total_io_time_usecs"`
	NumRandomIo                          string `json:"num_random_io"`
	HypervisorKvmMemoryUsagePpm          string `json:"hypervisor_kvm_memory_usage_ppm"`
	ControllerAvgWriteIoSizeKbytes       string `json:"controller_avg_write_io_size_kbytes"`
	ControllerAvgReadIoLatencyUsecs      string `json:"controller_avg_read_io_latency_usecs"`
	NumWriteIo                           string `json:"num_write_io"`
	HypervisorEsxCPUUsagePpm             string `json:"hypervisor_esx_cpu_usage_ppm"`
	TotalIoSizeKbytes                    string `json:"total_io_size_kbytes"`
	IoBandwidthKBps                      string `json:"io_bandwidth_kBps"`
	ContentCachePhysicalMemoryUsageBytes string `json:"content_cache_physical_memory_usage_bytes"`
	ReplicationNumReceivedBytes          string `json:"replication_num_received_bytes"`
	ControllerTimespanUsecs              string `json:"controller_timespan_usecs"`
	NumSeqIo                             string `json:"num_seq_io"`
	ContentCacheSavedMemoryUsageBytes    string `json:"content_cache_saved_memory_usage_bytes"`
	SeqIoPpm                             string `json:"seq_io_ppm"`
	WriteIoPpm                           string `json:"write_io_ppm"`
	ControllerAvgWriteIoLatencyUsecs     string `json:"controller_avg_write_io_latency_usecs"`
	ContentCacheLogicalMemoryUsageBytes  string `json:"content_cache_logical_memory_usage_bytes"`
}

type ClusterUsageStats struct {
	DataReductionOverallSavingRatioPpm           string `json:"data_reduction.overall.saving_ratio_ppm"`
	StorageReservedFreeBytes                     string `json:"storage.reserved_free_bytes"`
	StorageTierDasSataUsageBytes                 string `json:"storage_tier.das-sata.usage_bytes"`
	DataReductionCompressionSavedBytes           string `json:"data_reduction.compression.saved_bytes"`
	DataReductionSavingRatioPpm                  string `json:"data_reduction.saving_ratio_ppm"`
	DataReductionErasureCodingPostReductionBytes string `json:"data_reduction.erasure_coding.post_reduction_bytes"`
	StorageTierSsdPinnedUsageBytes               string `json:"storage_tier.ssd.pinned_usage_bytes"`
	StorageReservedUsageBytes                    string `json:"storage.reserved_usage_bytes"`
	DataReductionErasureCodingSavingRatioPpm     string `json:"data_reduction.erasure_coding.saving_ratio_ppm"`
	DataReductionThinProvisionSavedBytes         string `json:"data_reduction.thin_provision.saved_bytes"`
	StorageTierDasSataCapacityBytes              string `json:"storage_tier.das-sata.capacity_bytes"`
	StorageTierDasSataFreeBytes                  string `json:"storage_tier.das-sata.free_bytes"`
	StorageUsageBytes                            string `json:"storage.usage_bytes"`
	DataReductionErasureCodingSavedBytes         string `json:"data_reduction.erasure_coding.saved_bytes"`
	DataReductionCompressionPreReductionBytes    string `json:"data_reduction.compression.pre_reduction_bytes"`
	StorageRebuildCapacityBytes                  string `json:"storage.rebuild_capacity_bytes"`
	StorageTierDasSataPinnedUsageBytes           string `json:"storage_tier.das-sata.pinned_usage_bytes"`
	DataReductionPreReductionBytes               string `json:"data_reduction.pre_reduction_bytes"`
	StorageTierSsdCapacityBytes                  string `json:"storage_tier.ssd.capacity_bytes"`
	DataReductionCloneSavedBytes                 string `json:"data_reduction.clone.saved_bytes"`
	StorageTierSsdFreeBytes                      string `json:"storage_tier.ssd.free_bytes"`
	DataReductionDedupPreReductionBytes          string `json:"data_reduction.dedup.pre_reduction_bytes"`
	DataReductionErasureCodingPreReductionBytes  string `json:"data_reduction.erasure_coding.pre_reduction_bytes"`
	StorageCapacityBytes                         string `json:"storage.capacity_bytes"`
	DataReductionDedupPostReductionBytes         string `json:"data_reduction.dedup.post_reduction_bytes"`
	DataReductionCloneSavingRatioPpm             string `json:"data_reduction.clone.saving_ratio_ppm"`
	StorageLogicalUsageBytes                     string `json:"storage.logical_usage_bytes"`
	DataReductionSavedBytes                      string `json:"data_reduction.saved_bytes"`
	StorageFreeBytes                             string `json:"storage.free_bytes"`
	StorageTierSsdUsageBytes                     string `json:"storage_tier.ssd.usage_bytes"`
	DataReductionCompressionPostReductionBytes   string `json:"data_reduction.compression.post_reduction_bytes"`
	DataReductionPostReductionBytes              string `json:"data_reduction.post_reduction_bytes"`
	DataReductionDedupSavedBytes                 string `json:"data_reduction.dedup.saved_bytes"`
	DataReductionOverallSavedBytes               string `json:"data_reduction.overall.saved_bytes"`
	DataReductionThinProvisionPostReductionBytes string `json:"data_reduction.thin_provision.post_reduction_bytes"`
	DataReductionThinProvisionSavingRatioPpm     string `json:"data_reduction.thin_provision.saving_ratio_ppm"`
	DataReductionCompressionSavingRatioPpm       string `json:"data_reduction.compression.saving_ratio_ppm"`
	DataReductionDedupSavingRatioPpm             string `json:"data_reduction.dedup.saving_ratio_ppm"`
	StorageTierSsdPinnedBytes                    string `json:"storage_tier.ssd.pinned_bytes"`
	StorageReservedCapacityBytes                 string `json:"storage.reserved_capacity_bytes"`
	DataReductionThinProvisionPreReductionBytes  string `json:"data_reduction.thin_provision.pre_reduction_bytes"`
}

type SCluster struct {
	Id                   string `json:"id"`
	UUID                 string `json:"uuid"`
	ClusterIncarnationId int64  `json:"cluster_incarnation_id"`
	ClusterUUID          string `json:"cluster_uuid"`
	Name                 string `json:"name"`
	//ClusterExternalIpaddress              interface{}                        `json:"cluster_external_ipaddress"`
	//ClusterFullyQualifiedDomainName       interface{}                        `json:"cluster_fully_qualified_domain_name"`
	IsNsenabled bool `json:"is_nsenabled"`
	//ClusterExternalDataServicesIpaddress  interface{}                        `json:"cluster_external_data_services_ipaddress"`
	//SegmentedIscsiDataServicesIpaddress   interface{}                        `json:"segmented_iscsi_data_services_ipaddress"`
	//ClusterMasqueradingIpaddress          interface{}                        `json:"cluster_masquerading_ipaddress"`
	//ClusterMasqueradingPort               interface{}                        `json:"cluster_masquerading_port"`
	Timezone                              string   `json:"timezone"`
	SupportVerbosityType                  string   `json:"support_verbosity_type"`
	OperationMode                         string   `json:"operation_mode"`
	Encrypted                             bool     `json:"encrypted"`
	ClusterUsageWarningAlertThresholdPct  int      `json:"cluster_usage_warning_alert_threshold_pct"`
	ClusterUsageCriticalAlertThresholdPct int      `json:"cluster_usage_critical_alert_threshold_pct"`
	StorageType                           string   `json:"storage_type"`
	ClusterFunctions                      []string `json:"cluster_functions"`
	IsLts                                 bool     `json:"is_lts"`
	//IsRegisteredToPc                      interface{}                        `json:"is_registered_to_pc"`
	NumNodes                           int      `json:"num_nodes"`
	BlockSerials                       []string `json:"block_serials"`
	Version                            string   `json:"version"`
	FullVersion                        string   `json:"full_version"`
	TargetVersion                      string   `json:"target_version"`
	ExternalSubnet                     string   `json:"external_subnet"`
	InternalSubnet                     string   `json:"internal_subnet"`
	NccVersion                         string   `json:"ncc_version"`
	EnableLockDown                     bool     `json:"enable_lock_down"`
	EnablePasswordRemoteLoginToCluster bool     `json:"enable_password_remote_login_to_cluster"`
	FingerprintContentCachePercentage  int      `json:"fingerprint_content_cache_percentage"`
	SsdPinningPercentageLimit          int      `json:"ssd_pinning_percentage_limit"`
	EnableShadowClones                 bool     `json:"enable_shadow_clones"`
	//GlobalNfsWhiteList                 []interface{}                      `json:"global_nfs_white_list"`
	//NameServers                        []string                           `json:"name_servers"`
	//NtpServers                         []string                           `json:"ntp_servers"`
	//ServiceCenters                     []interface{}                      `json:"service_centers"`
	//HTTPProxies                        []interface{}                      `json:"http_proxies"`
	//RackableUnits                      []RackableUnits                    `json:"rackable_units"`
	//PublicKeys                         []interface{}                      `json:"public_keys"`
	//SMTPServer                         interface{}                        `json:"smtp_server"`
	HypervisorTypes                    []string                           `json:"hypervisor_types"`
	ClusterRedundancyState             ClusterRedundancyState             `json:"cluster_redundancy_state"`
	Multicluster                       bool                               `json:"multicluster"`
	Cloudcluster                       bool                               `json:"cloudcluster"`
	HasSelfEncryptingDrive             bool                               `json:"has_self_encrypting_drive"`
	IsUpgradeInProgress                bool                               `json:"is_upgrade_in_progress"`
	SecurityComplianceConfig           SecurityComplianceConfig           `json:"security_compliance_config"`
	HypervisorSecurityComplianceConfig HypervisorSecurityComplianceConfig `json:"hypervisor_security_compliance_config"`
	HypervisorLldpConfig               HypervisorLldpConfig               `json:"hypervisor_lldp_config"`
	ClusterArch                        string                             `json:"cluster_arch"`
	//IscsiConfig                        interface{}                        `json:"iscsi_config"`
	//Domain                             interface{}                        `json:"domain"`
	NosClusterAndHostsDomainJoined  bool `json:"nos_cluster_and_hosts_domain_joined"`
	AllHypervNodesInFailoverCluster bool `json:"all_hyperv_nodes_in_failover_cluster"`
	//Credential                        interface{}       `json:"credential"`
	Stats                             ClusterStats      `json:"stats"`
	UsageStats                        ClusterUsageStats `json:"usage_stats"`
	EnforceRackableUnitAwarePlacement bool              `json:"enforce_rackable_unit_aware_placement"`
	DisableDegradedNodeMonitoring     bool              `json:"disable_degraded_node_monitoring"`
	CommonCriteriaMode                bool              `json:"common_criteria_mode"`
	EnableOnDiskDedup                 bool              `json:"enable_on_disk_dedup"`
	//ManagementServers                 interface{}       `json:"management_servers"`
	FaultToleranceDomainType string `json:"fault_tolerance_domain_type"`
	//ThresholdForStorageThinProvision interface{} `json:"threshold_for_storage_thin_provision"`
}

func (self *SRegion) GetClusters() ([]SCluster, error) {
	clusters := []SCluster{}
	err := self.listAll("clusters", nil, &clusters)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func (self *SRegion) GetCluster(id string) (*SCluster, error) {
	cluster := &SCluster{}
	return cluster, self.client.get("clusters", id, nil, cluster)
}
