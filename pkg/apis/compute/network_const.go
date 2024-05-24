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
	"time"

	"yunion.io/x/cloudmux/pkg/apis/compute"
)

type TNetworkType string

const (
	// # DEFAULT_BANDWIDTH = options.default_bandwidth

	// in mbps, maximal is 100Tbps
	MAX_BANDWIDTH = 100000000

	NETWORK_TYPE_GUEST     = TNetworkType(compute.NETWORK_TYPE_GUEST)
	NETWORK_TYPE_BAREMETAL = TNetworkType("baremetal")
	NETWORK_TYPE_CONTAINER = TNetworkType("container")
	NETWORK_TYPE_PXE       = TNetworkType("pxe")
	NETWORK_TYPE_IPMI      = TNetworkType("ipmi")
	NETWORK_TYPE_EIP       = TNetworkType("eip")
	NETWORK_TYPE_HOSTLOCAL = TNetworkType("hostlocal")

	STATIC_ALLOC = "static"

	MAX_NETWORK_NAME_LEN = 11

	EXTRA_DNS_UPDATE_TARGETS = "__extra_dns_update_targets"

	NETWORK_STATUS_INIT          = "init"
	NETWORK_STATUS_PENDING       = compute.NETWORK_STATUS_PENDING
	NETWORK_STATUS_AVAILABLE     = compute.NETWORK_STATUS_AVAILABLE
	NETWORK_STATUS_UNAVAILABLE   = compute.NETWORK_STATUS_UNAVAILABLE
	NETWORK_STATUS_FAILED        = "failed"
	NETWORK_STATUS_UNKNOWN       = "unknown"
	NETWORK_STATUS_START_DELETE  = "start_delete"
	NETWORK_STATUS_DELETING      = compute.NETWORK_STATUS_DELETING
	NETWORK_STATUS_DELETED       = "deleted"
	NETWORK_STATUS_DELETE_FAILED = "delete_failed"
)

const (
	NETWORK_DRIVER_VIRTIO  = "virtio"
	NETWORK_DRIVER_E1000   = "e1000"
	NETWORK_DRIVER_VMXNET3 = "vmxnet3"
	NETWORK_DRIVER_VFIO    = "vfio-pci"
)

var (
	ALL_NETWORK_TYPES = []TNetworkType{
		NETWORK_TYPE_GUEST,
		NETWORK_TYPE_BAREMETAL,
		NETWORK_TYPE_CONTAINER,
		NETWORK_TYPE_PXE,
		NETWORK_TYPE_IPMI,
		NETWORK_TYPE_EIP,
		NETWORK_TYPE_HOSTLOCAL,
	}

	REGIONAL_NETWORK_PROVIDERS = []string{
		CLOUD_PROVIDER_HUAWEI,
		CLOUD_PROVIDER_HCSO,
		CLOUD_PROVIDER_HCSOP,
		CLOUD_PROVIDER_HCS,
		CLOUD_PROVIDER_CTYUN,
		CLOUD_PROVIDER_UCLOUD,
		CLOUD_PROVIDER_GOOGLE,
		CLOUD_PROVIDER_OPENSTACK,
		CLOUD_PROVIDER_JDCLOUD,
		CLOUD_PROVIDER_H3C,
	}
)

func IsInNetworkTypes(netType TNetworkType, types []TNetworkType) bool {
	for _, t := range types {
		if t == netType {
			return true
		}
	}
	return false
}

type IPAllocationDirection string

const (
	IPAllocationStepdown IPAllocationDirection = "stepdown"
	IPAllocationStepup   IPAllocationDirection = "stepup"
	IPAllocationRandom   IPAllocationDirection = "random"
	IPAllocationNone     IPAllocationDirection = "none"
	IPAllocationDefault                        = ""
)

type SNetworkUsedAddress struct {
	IpAddr        string
	Ip6Addr       string
	MacAddr       string
	Owner         string
	OwnerId       string
	OwnerStatus   string
	OwnerType     string
	AssociateId   string
	AssociateType string
	CreatedAt     time.Time
}
