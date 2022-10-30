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
	VPC_STATUS_PENDING     = "pending"
	VPC_STATUS_AVAILABLE   = "available"
	VPC_STATUS_UNAVAILABLE = "unavailable"
	VPC_STATUS_FAILED      = "failed"
	VPC_STATUS_DELETING    = "deleting"
	VPC_STATUS_UNKNOWN     = "unknown"

	DEFAULT_VPC_ID = "default"
	NORMAL_VPC_ID  = "normal" // 没有关联VPC的安全组，统一使用normal
)
