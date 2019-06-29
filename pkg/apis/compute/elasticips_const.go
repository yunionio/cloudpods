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
	EIP_MODE_INSTANCE_PUBLICIP = "public_ip"
	EIP_MODE_STANDALONE_EIP    = "elastic_ip"

	EIP_ASSOCIATE_TYPE_SERVER      = "server"
	EIP_ASSOCIATE_TYPE_NAT_GATEWAY = "natgateway"

	EIP_STATUS_READY           = "ready"
	EIP_STATUS_UNKNOWN         = "unknown"
	EIP_STATUS_ALLOCATE        = "allocate"
	EIP_STATUS_ALLOCATE_FAIL   = "allocate_fail"
	EIP_STATUS_DEALLOCATE      = "deallocate"
	EIP_STATUS_DEALLOCATE_FAIL = "deallocate_fail"
	EIP_STATUS_ASSOCIATE       = "associate"
	EIP_STATUS_ASSOCIATE_FAIL  = "associate_fail"
	EIP_STATUS_DISSOCIATE      = "dissociate"
	EIP_STATUS_DISSOCIATE_FAIL = "dissociate_fail"

	EIP_STATUS_CHANGE_BANDWIDTH = "change_bandwidth"

	EIP_CHARGE_TYPE_BY_TRAFFIC   = "traffic"
	EIP_CHARGE_TYPE_BY_BANDWIDTH = "bandwidth"
	EIP_CHARGE_TYPE_DEFAULT      = EIP_CHARGE_TYPE_BY_TRAFFIC

	// EIP associate resource type
	EIP_ASSOCIATE_TYPE_VM  = "SERVER"
	EIP_ASSOCIATE_TYPE_ELB = "ELB"
)

var (
	EIP_ASSOCIATE_VALID_TYPES = []string{EIP_ASSOCIATE_TYPE_SERVER, EIP_ASSOCIATE_TYPE_NAT_GATEWAY}
)
