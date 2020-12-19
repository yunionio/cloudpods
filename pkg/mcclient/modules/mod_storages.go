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

package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	Storages modulebase.ResourceManager
)

func init() {
	Storages = NewComputeManager("storage", "storages",
		[]string{"ID", "Name", "Capacity", "Actual_capacity_used", "Status", "Used_capacity", "Waste_capacity", "Free_capacity", "Storage_type", "Medium_type", "Virtual_capacity", "commit_bound", "commit_rate", "Enabled", "public_scope"},
		[]string{})

	registerCompute(&Storages)
}
