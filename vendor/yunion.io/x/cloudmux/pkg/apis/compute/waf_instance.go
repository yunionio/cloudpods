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
	WAF_ACTION_ALLOW      = "Allow"
	WAF_ACTION_BLOCK      = "Block"
	WAF_ACTION_PREVENTION = "Prevention"
	WAF_ACTION_DETECTION  = "Detection"

	WAF_STATUS_AVAILABLE     = "available"
	WAF_STATUS_DELETING      = "deleting"
	WAF_STATUS_DELETE_FAILED = "delete_failed"
	WAF_STATUS_CREATING      = "creating"
	WAF_STATUS_CREATE_FAILED = "create_failed"
	WAF_STATUS_UPDATING      = "updating"
	WAF_STATUS_UNKNOWN       = "unknown"
)
