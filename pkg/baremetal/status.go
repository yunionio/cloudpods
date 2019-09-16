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

package baremetal

const (
	INIT           = "init"
	PREPARE        = "prepare"
	PREPARE_FAIL   = "prepare_fail"
	READY          = "ready"
	RUNNING        = "running"
	MAINTAINING    = "maintaining"
	START_MAINTAIN = "start_maintain"
	DELETING       = "deleting"
	DELETE         = "delete"
	DELETE_FAIL    = "delete_fail"
	UNKNOWN        = "unknown"
	SYNCING_STATUS = "syncing_status"
	SYNC           = "sync"
	SYNC_FAIL      = "sync_fail"
	START_CONVERT  = "start_convert"
	CONVERTING     = "converting"
	START_FAIL     = "start_fail"
	STOP_FAIL      = "stop_fail"
)
