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

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

func init() {
	type WorkerListOptions struct {
		ServiceType string `choices:"image|cloudid|cloudevent|devtool|ansible|identity|notify|log|compute|compute_v2|meter|suggestion|cloudmon|cloudproxy|dns|monitor|scheduler|vpcagent|webconsole|yunionconf|bastionhost|extdb|yunionapi"`
	}
	R(&WorkerListOptions{}, "worker-list", "List workers", func(s *mcclient.ClientSession, args *WorkerListOptions) error {
		result, err := modules.Workers.List(s, jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printList(result, modules.Workers.GetColumns(s))
		return nil
	})
}
