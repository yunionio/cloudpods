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
	ST_TYPE_TIMING = "timing" // 定时
	ST_TYPE_CYCLE  = "cycle"  // 周期

	ST_STATUS_READY         = "ready"
	ST_STATUS_CREATE_FAILED = "create_failed"

	ST_RESOURCE_SERVER = "server"

	ST_RESOURCE_OPERATION_START   = "start"
	ST_RESOURCE_OPERATION_STOP    = "stop"
	ST_RESOURCE_OPERATION_RESTART = "restart"

	ST_LABEL_ID  = "id"
	ST_LABEL_TAG = "tag"

	ST_ACTIVITY_STATUS_EXEC         = "execution"    // 执行中
	ST_ACTIVITY_STATUS_SUCCEED      = "succeed"      // 成功
	ST_ACTIVITY_STATUS_PART_SUCCEED = "part_succeed" // 部分成功
	ST_ACTIVITY_STATUS_FAILED       = "failed"       // 失败
	ST_ACTIVITY_STATUS_REJECT       = "reject"       // 拒绝
)
