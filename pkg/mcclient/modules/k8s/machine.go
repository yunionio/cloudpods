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

package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	KubeMachines *ResourceManager
)

func init() {
	KubeMachines = NewResourceManager("kubemachine", "kubemachines",
		NewResourceCols("Role", "First_Node", "Cluster", "Provider", "Resource_Type", "Resource_Id", "Status", "Address", "Hypervisor", "Zone_Id", "Network_Id"),
		NewColumns())
	modules.Register(KubeMachines)
}
