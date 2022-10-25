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
	"yunion.io/x/onecloud/pkg/multicloud/hcs"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NetworkListOptions struct {
		Vpc string `help:"Vpc ID"`
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List subnets", func(cli *hcs.SRegion, args *NetworkListOptions) error {
		networks, e := cli.GetNetwroks(args.Vpc)
		if e != nil {
			return e
		}
		printList(networks, 0, 0, 0, nil)
		return nil
	})

	type NetworkIdOptions struct {
		ID string
	}

	shellutils.R(&NetworkIdOptions{}, "network-show", "Show subnets", func(cli *hcs.SRegion, args *NetworkIdOptions) error {
		network, e := cli.GetNetwork(args.ID)
		if e != nil {
			return e
		}
		printObject(network)
		return nil
	})

}
