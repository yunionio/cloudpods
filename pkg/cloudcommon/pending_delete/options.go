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

package pending_delete

type SPendingDeleteOptions struct {
	EnablePendingDelete            bool `default:"true" help:"Turn on/off pending-delete resource, default is on" json:"enable_pending_delete"`
	PendingDeleteCheckSeconds      int  `default:"3600" help:"How long to wait to scan pending-delete resource, default is 1 hour"`
	PendingDeleteExpireSeconds     int  `default:"259200" help:"How long a pending-delete resource cleaned automatically, default 3 days" alias:"scrub_time"`
	PendingDeleteMaxCleanBatchSize int  `default:"50" help:"How many pending-delete items can be clean in a batch"`
}
