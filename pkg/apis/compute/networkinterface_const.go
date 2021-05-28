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
	NETWORK_INTERFACE_ASSOCIATE_TYPE_SERVER       = "server"
	NETWORK_INTERFACE_ASSOCIATE_TYPE_RESERVED     = "reserved"
	NETWORK_INTERFACE_ASSOCIATE_TYPE_LOADBALANCER = "loadbalancer"
	NETWORK_INTERFACE_ASSOCIATE_TYPE_VIP          = "vip"
	NETWORK_INTERFACE_ASSOCIATE_TYPE_DHCP         = "dhcp"

	NETWORK_INTERFACE_STATUS_INIT      = "init"
	NETWORK_INTERFACE_STATUS_CREATING  = "creating"
	NETWORK_INTERFACE_STATUS_AVAILABLE = "available"
	NETWORK_INTERFACE_STATUS_DISABLED  = "disabled"
	NETWORK_INTERFACE_STATUS_ATTACHING = "attaching"
	NETWORK_INTERFACE_STATUS_DETACHING = "detaching"
	NETWORK_INTERFACE_STATUS_DELETING  = "deleting"
	NETWORK_INTERFACE_STATUS_UNKNOWN   = "unknown"
)
