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

const (
	HOST_TYPE_BAREMETAL  = "baremetal"
	HOST_TYPE_HYPERVISOR = "hypervisor" // KVM
	HOST_TYPE_KVM        = "kvm"
	HOST_TYPE_ESXI       = "esxi"    // # VMWare vSphere ESXi
	HOST_TYPE_KUBELET    = "kubelet" // # Kubernetes Kubelet
	HOST_TYPE_HYPERV     = "hyperv"  // # Microsoft Hyper-V
	HOST_TYPE_XEN        = "xen"     // # XenServer

	HOST_TYPE_ALIYUN    = "aliyun"
	HOST_TYPE_APSARA    = "apsara"
	HOST_TYPE_AWS       = "aws"
	HOST_TYPE_QCLOUD    = "qcloud"
	HOST_TYPE_AZURE     = "azure"
	HOST_TYPE_HUAWEI    = "huawei"
	HOST_TYPE_HCSO      = "hcso"
	HOST_TYPE_OPENSTACK = "openstack"
	HOST_TYPE_UCLOUD    = "ucloud"
	HOST_TYPE_ZSTACK    = "zstack"
	HOST_TYPE_GOOGLE    = "google"
	HOST_TYPE_CTYUN     = "ctyun"
	HOST_TYPE_ECLOUD    = "ecloud"
	HOST_TYPE_JDCLOUD   = "jdcloud"
	HOST_TYPE_CLOUDPODS = "cloudpods"

	HOST_TYPE_DEFAULT = HOST_TYPE_HYPERVISOR

	// # possible status
	HOST_ONLINE   = "online"
	HOST_ENABLED  = "online"
	HOST_OFFLINE  = "offline"
	HOST_DISABLED = "offline"

	NIC_TYPE_IPMI  = "ipmi"
	NIC_TYPE_ADMIN = "admin"
	// #NIC_TYPE_NORMAL = 'normal'

	BAREMETAL_INIT           = "init"
	BAREMETAL_PREPARE        = "prepare"
	BAREMETAL_PREPARE_FAIL   = "prepare_fail"
	BAREMETAL_READY          = "ready"
	BAREMETAL_RUNNING        = "running"
	BAREMETAL_MAINTAINING    = "maintaining"
	BAREMETAL_START_MAINTAIN = "start_maintain"
	BAREMETAL_MAINTAIN_FAIL  = "maintain_fail"
	BAREMETAL_DELETING       = "deleting"
	BAREMETAL_DELETE         = "delete"
	BAREMETAL_DELETE_FAIL    = "delete_fail"
	BAREMETAL_UNKNOWN        = "unknown"
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
	HOST_TYPE_OPENSTACK,
	HOST_TYPE_UCLOUD,
	HOST_TYPE_ZSTACK,
	HOST_TYPE_CTYUN,
	HOST_TYPE_GOOGLE,
	HOST_TYPE_JDCLOUD,
	HOST_TYPE_CLOUDPODS,
}

var NIC_TYPES = []string{NIC_TYPE_IPMI, NIC_TYPE_ADMIN}

const (
	ACCESS_MAC_ANY = "00:00:00:00:00:00"
)

const (
	BOOT_MODE_PXE = "pxe"
	BOOT_MODE_ISO = "iso"
)

const (
	HOST_HEALTH_PREFIX         = "/onecloud/kvm/host/health"
	HOST_HEALTH_STATUS_RUNNING = "running"
	HOST_HEALTH_LOCK_PREFIX    = "host-health"
)
