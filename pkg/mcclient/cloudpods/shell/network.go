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

package shell

import (
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/onecloud/pkg/mcclient/cloudpods"
)

func init() {
	type NetworkListOptions struct {
		WireId string
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *cloudpods.SRegion, args *NetworkListOptions) error {
		networks, err := cli.GetNetworks(args.WireId)
		if err != nil {
			return err
		}
		printList(networks, 0, 0, 0, nil)
		return nil
	})
}
