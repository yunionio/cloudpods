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
	"yunion.io/x/cloudmux/pkg/apis/compute"
)

const (
	HOST_TYPE_BAREMETAL  = "baremetal"
	HOST_TYPE_HYPERVISOR = "hypervisor" // KVM
	HOST_TYPE_KVM        = "kvm"
	HOST_TYPE_ESXI       = compute.HOST_TYPE_ESXI // # VMWare vSphere ESXi
	HOST_TYPE_KUBELET    = "kubelet"              // # Kubernetes Kubelet
	HOST_TYPE_HYPERV     = "hyperv"               // # Microsoft Hyper-V
	HOST_TYPE_XEN        = "xen"                  // # XenServer

	HOST_TYPE_ALIYUN         = compute.HOST_TYPE_ALIYUN
	HOST_TYPE_APSARA         = compute.HOST_TYPE_APSARA
	HOST_TYPE_AWS            = compute.HOST_TYPE_AWS
	HOST_TYPE_QCLOUD         = compute.HOST_TYPE_QCLOUD
	HOST_TYPE_AZURE          = compute.HOST_TYPE_AZURE
	HOST_TYPE_HUAWEI         = compute.HOST_TYPE_HUAWEI
	HOST_TYPE_HCSO           = compute.HOST_TYPE_HCSO
	HOST_TYPE_HCSOP          = compute.HOST_TYPE_HCSOP
	HOST_TYPE_HCS            = compute.HOST_TYPE_HCS
	HOST_TYPE_OPENSTACK      = compute.HOST_TYPE_OPENSTACK
	HOST_TYPE_UCLOUD         = compute.HOST_TYPE_UCLOUD
	HOST_TYPE_VOLCENGINE     = compute.HOST_TYPE_VOLCENGINE
	HOST_TYPE_ZSTACK         = compute.HOST_TYPE_ZSTACK
	HOST_TYPE_GOOGLE         = compute.HOST_TYPE_GOOGLE
	HOST_TYPE_CTYUN          = compute.HOST_TYPE_CTYUN
	HOST_TYPE_ECLOUD         = compute.HOST_TYPE_ECLOUD
	HOST_TYPE_JDCLOUD        = compute.HOST_TYPE_JDCLOUD
	HOST_TYPE_CLOUDPODS      = compute.HOST_TYPE_CLOUDPODS
	HOST_TYPE_NUTANIX        = compute.HOST_TYPE_NUTANIX
	HOST_TYPE_BINGO_CLOUD    = compute.HOST_TYPE_BINGO_CLOUD
	HOST_TYPE_INCLOUD_SPHERE = compute.HOST_TYPE_INCLOUD_SPHERE
	HOST_TYPE_PROXMOX        = compute.HOST_TYPE_PROXMOX
	HOST_TYPE_REMOTEFILE     = compute.HOST_TYPE_REMOTEFILE
	HOST_TYPE_H3C            = compute.HOST_TYPE_H3C
	HOST_TYPE_KSYUN          = compute.HOST_TYPE_KSYUN
	HOST_TYPE_BAIDU          = compute.HOST_TYPE_BAIDU
	HOST_TYPE_CUCLOUD        = compute.HOST_TYPE_CUCLOUD
	HOST_TYPE_QINGCLOUD      = compute.HOST_TYPE_QINGCLOUD
	HOST_TYPE_ORACLE         = compute.HOST_TYPE_ORACLE

	HOST_TYPE_DEFAULT = HOST_TYPE_HYPERVISOR

	// # possible status
	HOST_ONLINE   = compute.HOST_ONLINE
	HOST_ENABLED  = "online"
	HOST_OFFLINE  = compute.HOST_OFFLINE
	HOST_DISABLED = "offline"

	NIC_TYPE_IPMI       = compute.NIC_TYPE_IPMI
	NIC_TYPE_ADMIN      = compute.NIC_TYPE_ADMIN
	NIC_TYPE_NORMAL     = compute.NIC_TYPE_NORMAL
	NIC_TYPE_INFINIBAND = compute.TNicType("infiniband")

	BAREMETAL_INIT           = "init"
	BAREMETAL_PREPARE        = "prepare"
	BAREMETAL_PREPARE_FAIL   = "prepare_fail"
	BAREMETAL_READY          = compute.BAREMETAL_READY
	BAREMETAL_RUNNING        = compute.BAREMETAL_RUNNING
	BAREMETAL_MAINTAINING    = "maintaining"
	BAREMETAL_START_MAINTAIN = "start_maintain"
	BAREMETAL_MAINTAIN_FAIL  = "maintain_fail"
	BAREMETAL_DELETING       = "deleting"
	BAREMETAL_DELETE         = "delete"
	BAREMETAL_DELETE_FAIL    = "delete_fail"
	BAREMETAL_UNKNOWN        = compute.BAREMETAL_UNKNOWN
	BAREMETAL_SYNCING_STATUS = "syncing_status"
	BAREMETAL_SYNC           = "sync"
	BAREMETAL_SYNC_FAIL      = "sync_fail"
	BAREMETAL_START_CONVERT  = "start_convert"
	BAREMETAL_CONVERTING     = "converting"
	BAREMETAL_START_FAIL     = "start_fail"
	BAREMETAL_STOP_FAIL      = "stop_fail"

	BAREMETAL_START_PROBE = "start_probe"
	BAREMETAL_PROBING     = "probing"
	BAREMETAL_PROBE_FAIL  = "probe_fail"

	BAREMETAL_START_INSERT_ISO = "start_insert"
	BAREMETAL_INSERTING_ISO    = "inserting_iso"
	BAREMETAL_INSERT_FAIL      = "insert_fail"

	BAREMETAL_START_EJECT_ISO = "start_eject"
	BAREMETAL_EJECTING_ISO    = "ejecting_iso"
	BAREMETAL_EJECT_FAIL      = "eject_fail"

	HOST_STATUS_RUNNING = BAREMETAL_RUNNING
	HOST_STATUS_READY   = BAREMETAL_READY
	HOST_STATUS_UNKNOWN = BAREMETAL_UNKNOWN
)

const (
	BAREMETAL_CDROM_ACTION_INSERT = "insert"
	BAREMETAL_CDROM_ACTION_EJECT  = "eject"
)

const (
	HostResourceTypeShared         = "shared"
	HostResourceTypeDefault        = HostResourceTypeShared
	HostResourceTypePrepaidRecycle = "prepaid"
	HostResourceTypeDedicated      = "dedicated"
)

const (
	HOST_METADATA_CPU_USAGE_PERCENT = "cpu_usage_percent"
	HOST_METADATA_MEMORY_USED_MB    = "memory_used_mb"
)

var HOST_TYPES = []string{
	HOST_TYPE_BAREMETAL,
	HOST_TYPE_HYPERVISOR,
	HOST_TYPE_ESXI,
	HOST_TYPE_KUBELET,
	HOST_TYPE_XEN,
	HOST_TYPE_ALIYUN,
	HOST_TYPE_APSARA,
	HOST_TYPE_AZURE,
	HOST_TYPE_AWS,
	HOST_TYPE_QCLOUD,
	HOST_TYPE_HUAWEI,
	HOST_TYPE_HCSO,
	HOST_TYPE_HCS,
	HOST_TYPE_HCSOP,
	HOST_TYPE_OPENSTACK,
	HOST_TYPE_UCLOUD,
	HOST_TYPE_VOLCENGINE,
	HOST_TYPE_ZSTACK,
	HOST_TYPE_CTYUN,
	HOST_TYPE_GOOGLE,
	HOST_TYPE_JDCLOUD,
	HOST_TYPE_CLOUDPODS,
	HOST_TYPE_NUTANIX,
	HOST_TYPE_BINGO_CLOUD,
	HOST_TYPE_INCLOUD_SPHERE,
	HOST_TYPE_PROXMOX,
	HOST_TYPE_REMOTEFILE,
	HOST_TYPE_H3C,
	HOST_TYPE_KSYUN,
	HOST_TYPE_BAIDU,
	HOST_TYPE_CUCLOUD,
	HOST_TYPE_QINGCLOUD,
	HOST_TYPE_ORACLE,
}

var ALL_NIC_TYPES = []compute.TNicType{NIC_TYPE_IPMI, NIC_TYPE_ADMIN, NIC_TYPE_NORMAL}
var HOST_NIC_TYPES = []compute.TNicType{NIC_TYPE_ADMIN, NIC_TYPE_NORMAL}

const (
	ACCESS_MAC_ANY = "00:00:00:00:00:00"
)

const (
	BOOT_MODE_PXE = "pxe"
	BOOT_MODE_ISO = "iso"
)

const (
	HOST_HEALTH_PREFIX              = "/onecloud/kvm/host/health"
	HOST_HEALTH_STATUS_RUNNING      = "running"
	HOST_HEALTH_STATUS_RECONNECTING = "reconnecting"
	HOST_HEALTH_STATUS_UNKNOWN      = "unknown"
	HOST_HEALTH_LOCK_PREFIX         = "host-health"
)

const (
	HOSTMETA_AUTO_MIGRATE_ON_HOST_DOWN     = "__auto_migrate_on_host_down"
	HOSTMETA_AUTO_MIGRATE_ON_HOST_SHUTDOWN = "__auto_migrate_on_host_shutdown"
	HOSTMETA_HOST_ERRORS                   = "__host_errors"
)

const (
	HOSTMETA_RESERVED_CPUS_INFO = "reserved_cpus_info"
)
