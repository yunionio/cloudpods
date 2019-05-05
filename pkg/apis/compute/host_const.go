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
	HOST_TYPE_AWS       = "aws"
	HOST_TYPE_QCLOUD    = "qcloud"
	HOST_TYPE_AZURE     = "azure"
	HOST_TYPE_HUAWEI    = "huawei"
	HOST_TYPE_OPENSTACK = "openstack"
	HOST_TYPE_UCLOUD    = "ucloud"

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

	HOST_STATUS_RUNNING = BAREMETAL_RUNNING
	HOST_STATUS_READY   = BAREMETAL_READY
	HOST_STATUS_UNKNOWN = BAREMETAL_UNKNOWN
)

const (
	HostResourceTypeShared         = "shared"
	HostResourceTypeDefault        = HostResourceTypeShared
	HostResourceTypePrepaidRecycle = "prepaid"
	HostResourceTypeDedicated      = "dedicated"
)

var HOST_TYPES = []string{HOST_TYPE_BAREMETAL, HOST_TYPE_HYPERVISOR, HOST_TYPE_ESXI, HOST_TYPE_KUBELET, HOST_TYPE_XEN, HOST_TYPE_ALIYUN, HOST_TYPE_AZURE, HOST_TYPE_AWS, HOST_TYPE_QCLOUD, HOST_TYPE_HUAWEI, HOST_TYPE_OPENSTACK, HOST_TYPE_UCLOUD}

var NIC_TYPES = []string{NIC_TYPE_IPMI, NIC_TYPE_ADMIN}
