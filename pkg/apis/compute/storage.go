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

package compute

import (
	"strconv"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

type StorageUsage struct {
	HostCount     int
	DiskCount     int
	SnapshotCount int
	Used          int64
	Wasted        int64
}

func (self StorageUsage) IsZero() bool {
	return self.HostCount+self.DiskCount+self.SnapshotCount == 0
}

type StorageHardwareInfo struct {
	Model     *string `json:"model"`
	Vendor    *string `json:"vendor"`
	Bandwidth float64 `json:"bandwidth" help:"Bandwidth of the device, and the unit is GB/s"`
}

type StorageCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	// 存储类型
	//
	//
	//
	// | storage_type    | 参数                        |是否必传    |    默认值    | 说明        |
	// | --------        | -------                    | --------    | --------    | ---------    |
	// | rbd            | rbd_mon_host                | 是        |            | ceph mon_host    |
	// | rbd             | rbd_pool                    | 是         |            | ceph pool        |
	// | rbd             | rbd_key                    | 否         |            |若cephx认证开启,此参数必传    |
	// | rbd             | rbd_rados_mon_op_timeout    | 否         |    3        |单位: 秒    |
	// | rbd             | rbd_rados_osd_op_timeout    | 否         |    1200    |单位: 秒    |
	// | rbd             | rbd_client_mount_timeout    | 否         |    120        |单位: 秒    |
	// | nfs             | nfs_host                    | 是         |            |网络文件系统主机    |
	// | nfs             | nfs_shared_dir            | 是         |            |网络文件系统共享目录    |
	// local: 本地存储
	// rbd: ceph块存储, ceph存储创建时仅会检测是否重复创建，不会具体检测认证参数是否合法，只有挂载存储时
	// 计算节点会验证参数，若挂载失败，宿主机和存储不会关联，可以通过查看存储日志查找挂载失败原因
	// enum: local, rbd, nfs, gpfs
	// required: true
	StorageType string `json:"storage_type"`

	// 存储介质类型
	// enum: rotate, ssd, hybird
	// required: true
	// default: ssd
	MediumType string `json:"medium_type"`

	ZoneResourceInput

	// ceph认证主机, storage_type为 rbd 时,此参数为必传项
	// 单个ip或以逗号分隔的多个ip具体可查询 /etc/ceph/ceph.conf 文件
	// example: 192.168.222.3,192.168.222.4,192.168.222.99
	RbdMonHost string `json:"rbd_mon_host"`

	// swagger:ignore
	MonHost string

	// ceph使用的pool, storage_type为 rbd 时,此参数为必传项
	// example: rbd
	RbdPool string `json:"rbd_pool"`

	// swagger:ignore
	Pool string

	// ceph集群密码,若ceph集群开启cephx认证,此参数必传
	// 可在ceph集群主机的/etc/ceph/ceph.client.admin.keyring文件中找到
	// example: AQDigB9dtnDAKhAAxS6X4zi4BPR/lIle4nf4Dw==
	RbdKey string `json:"rbd_key"`

	// swagger:ignore
	Key string

	RbdTimeoutInput

	// swagger:ignore
	ClientMountTimeout int

	// swagger:ignore
	StorageConf *jsonutils.JSONDict

	// 网络文件系统主机, storage_type 为 nfs 时,此参数必传
	// example: 192.168.222.2
	NfsHost string `json:"nfs_host"`

	// 网络文件系统共享目录, storage_type 为 nfs 时, 此参数必传
	// example: /nfs_root/
	NfsSharedDir string `json:"nfs_shared_dir"`

	// swagger:ignore
	HardwareInfo *StorageHardwareInfo `json:"hardware_info"`
	// CLVM VG Name
	CLVMVgName string
	// SLVM VG Name
	SLVMVgName string
	MasterHost string
}

type RbdTimeoutInput struct {
	// ceph集群连接超时时间, 单位秒
	// default: 3
	RbdRadosMonOpTimeout int `json:"rbd_rados_mon_op_timeout"`

	// swagger:ignore
	RadosMonOpTimeout int

	// ceph osd 操作超时时间, 单位秒
	// default: 1200
	RbdRadosOsdOpTimeout int `json:"rbd_rados_osd_op_timeout"`

	// swagger:ignore
	RadosOsdOpTimeout int

	// ceph CephFS挂载超时时间, 单位秒
	// default: 120
	RbdClientMountTimeout int `json:"rbd_client_mount_timeout"`
}

type SStorageCapacityInfo struct {
	// 已使用容量大小
	UsedCapacity int64 `json:"used_capacity"`
	// 浪费容量大小(异常磁盘大小总和)
	WasteCapacity int64 `json:"waste_capacity"`
	// 虚拟容量大小
	VirtualCapacity int64 `json:"virtual_capacity"`
	// 超分率
	CommitRate float64 `json:"commit_rate"`
	// 可使用容量
	FreeCapacity int64 `json:"free_capacity"`
}

type StorageHost struct {
	Id         string
	Name       string
	Status     string
	HostStatus string
}

type StorageDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails
	ManagedResourceInfo
	ZoneResourceInfo

	SStorage

	SStorageCapacityInfo
	ActualUsed int64 `json:"real_time_used_capacity,omitzero"`
	VCapacity  int64 `json:"virtual_capacity,omitzero"`

	Hosts []StorageHost `json:"hosts"`

	Schedtags []SchedtagShortDescDetails `json:"schedtags"`

	StorageUsage `json:"storage_usage"`

	// 超分比
	CommitBound float32 `json:"commit_bound"`
}

func (self StorageDetails) GetMetricTags() map[string]string {
	ret := map[string]string{
		"id":             self.Id,
		"storage_id":     self.Id,
		"storage_name":   self.Name,
		"brand":          self.Brand,
		"domain_id":      self.DomainId,
		"project_domain": self.ProjectDomain,
		"external_id":    self.ExternalId,
	}
	return AppendMetricTags(ret, self.MetadataResourceInfo)
}

func (self StorageDetails) GetMetricPairs() map[string]string {
	usageActive := "0"
	if self.Capacity > 0 {
		usageActive = strconv.FormatFloat(float64(self.ActualCapacityUsed)/float64(self.Capacity)*100.0, 'f', -1, 64)
	}
	ret := map[string]string{
		"free":         strconv.FormatFloat(float64(self.Capacity-self.ActualCapacityUsed), 'f', 2, 64),
		"usage_active": usageActive,
	}
	return ret
}

type StorageResourceInfo struct {
	// 归属云订阅ID
	ManagerId string `json:"manager_id"`

	ManagedResourceInfo

	// 归属可用区ID
	ZoneId string `json:"zone_id"`

	ZoneResourceInfo

	// 存储名称
	Storage string `json:"storage"`

	// 存储类型
	StorageType string `json:"storage_type"`

	// 存储介质类型
	MediumType string `json:"medium_type"`

	// 存储状态
	StorageStatus string `json:"storage_status"`
}

type StorageUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput

	// ceph集群密码,若ceph集群开启cephx认证,此参数必传
	// 可在ceph集群主机的/etc/ceph/ceph.client.admin.keyring文件中找到
	// example: AQDigB9dtnDAKhAAxS6X4zi4BPR/lIle4nf4Dw==
	RbdKey string `json:"rbd_key"`

	RbdTimeoutInput

	// swagger:ignore
	StorageConf *jsonutils.JSONDict

	UpdateStorageConf bool

	// swagger:ignore
	HardwareInfo *StorageHardwareInfo `json:"hardware_info"`
	MasterHost   string
}
