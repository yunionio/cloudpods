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

package misc

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/apimap"
	"yunion.io/x/onecloud/pkg/mcclient/modules/vpcagent"
)

func init() {
	type VpcAgentSyncOptions struct {
	}
	R(&VpcAgentSyncOptions{}, "vpcagent-sync", "Invoke sync of vpcagent", func(s *mcclient.ClientSession, args *VpcAgentSyncOptions) error {
		// when the platform serves the model sets from apimap, vpcagent only
		// picks them up there, so the apimap side has to be synced first
		if apimap.TriggerSyncAndWait(s) == apimap.SyncOutcomeUnchanged {
			fmt.Println("model sets did not change, skip vpcagent sync")
			return nil
		}
		err := vpcagent.VpcAgent.DoSync(s)
		if err != nil {
			return err
		}
		return nil
	})
}
