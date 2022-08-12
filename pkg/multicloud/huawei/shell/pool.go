// @@ -0,0 +1,46 @@
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
	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ModelartsPoolListOption struct {
		PoolName string `help:"Pool Name"`
	}
	shellutils.R(&ModelartsPoolListOption{}, "modelarts-pool-list", "List Modelarts Pool", func(cli *huawei.SRegion, args *ModelartsPoolListOption) error {
		pools, err := cli.GetPools()
		if err != nil {
			return err
		}
		printList(pools, len(pools), 0, 0, nil)
		return nil
	})

	// shellutils.R(&PoolListOption{}, "pool-detail", "List pool", func(cli *huawei.SRegion, args *PoolListOption) error {
	// 	pools, err := cli.GetPoolsByName(args.PoolName)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	printList(pools, len(pools), 0, 0, nil)
	// 	return nil
	// })

	/*type RouteTableIdOptions struct {
		ID string `help:"RouteTable ID"`
	}
	shellutils.R(&RouteTableIdOptions{}, "routetable-show", "Show vpc route table", func(cli *huawei.SRegion, args *RouteTableIdOptions) error {
		routetable, err := cli.GetRouteTable(args.ID)
		if err != nil {
			return err
		}
		printObject(routetable)
		return nil
	})*/

}
