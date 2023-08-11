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
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initKubeMachine() {
	cmd := NewK8sResourceCmd(k8s.KubeMachines)
	cmd.SetKeyword("machine")
	cmd.ShowEvent()
	cmd.List(new(o.MachineListOptions))
	cmd.Create(new(o.MachineCreateOptions))
	cmd.Show(new(o.IdentOptions))
	cmd.BatchDelete(new(o.IdentsOptions))
	cmd.Perform("recreate", new(o.IdentOptions))
	cmd.Perform("terminate", new(o.IdentOptions))
	cmd.Get("networkaddress", new(o.MachineListNetworkAddressOptions))
	cmd.Perform("attach-networkaddress", new(o.MachineAttachNetworkAddressOptions))
	cmd.Perform("post-prepare-resource", new(o.IdentOptions))
}
