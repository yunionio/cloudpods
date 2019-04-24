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
	// # DEFAULT_BANDWIDTH = options.default_bandwidth
	MAX_BANDWIDTH = 100000

	NETWORK_TYPE_GUEST     = "guest"
	NETWORK_TYPE_BAREMETAL = "baremetal"
	NETWORK_TYPE_CONTAINER = "container"
	NETWORK_TYPE_PXE       = "pxe"
	NETWORK_TYPE_IPMI      = "ipmi"

	STATIC_ALLOC = "static"

	MAX_NETWORK_NAME_LEN = 11

	EXTRA_DNS_UPDATE_TARGETS = "__extra_dns_update_targets"

	NETWORK_STATUS_INIT          = "init"
	NETWORK_STATUS_PENDING       = "pending"
	NETWORK_STATUS_AVAILABLE     = "available"
	NETWORK_STATUS_FAILED        = "failed"
	NETWORK_STATUS_UNKNOWN       = "unknown"
	NETWORK_STATUS_START_DELETE  = "start_delete"
	NETWORK_STATUS_DELETING      = "deleting"
	NETWORK_STATUS_DELETED       = "deleted"
	NETWORK_STATUS_DELETE_FAILED = "delete_failed"
)

var (
	ALL_NETWORK_TYPES = []string{
		NETWORK_TYPE_GUEST,
		NETWORK_TYPE_BAREMETAL,
		NETWORK_TYPE_CONTAINER,
		NETWORK_TYPE_PXE,
		NETWORK_TYPE_IPMI,
	}
)
