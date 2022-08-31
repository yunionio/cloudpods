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
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ModelartsPoolListOption struct {
		PoolId string `help:"Pool Id"`
	}

	shellutils.R(&ModelartsPoolListOption{}, "modelarts-pool-list", "List Modelarts Pool", func(cli *huawei.SRegion, args *ModelartsPoolListOption) error {
		pools, err := cli.GetClient().GetIModelartsPools()
		if err != nil {
			return err
		}
		printList(pools, len(pools), 0, 0, nil)
		return nil
	})

	shellutils.R(&ModelartsPoolListOption{}, "modelarts-pool-detail", "List pool", func(cli *huawei.SRegion, args *ModelartsPoolListOption) error {
		pools, err := cli.GetClient().GetIModelartsPoolById(args.PoolId)
		if err != nil {
			return err
		}
		log.Infoln(pools)
		return nil
	})

	shellutils.R(&cloudprovider.ModelartsPoolCreateOption{}, "modelarts-pool-create", "Create Modelarts Pool", func(cli *huawei.SRegion, args *cloudprovider.ModelartsPoolCreateOption) error {
		res, err := cli.GetClient().CreateIModelartsPool(args)
		if err != nil {
			return err
		}
		// printList(pools, len(pools), 0, 0, nil)
		log.Infoln("this is res:", res)
		return nil
	})

	shellutils.R(&ModelartsPoolListOption{}, "modelarts-pool-delete", "Delete Modelarts Pool", func(cli *huawei.SRegion, args *ModelartsPoolListOption) error {
		res, err := cli.GetClient().DeletePool(args.PoolId)
		if err != nil {
			return err
		}
		// printList(pools, len(pools), 0, 0, nil)
		log.Infoln("this is res:", res)
		return nil
	})
	shellutils.R(&cloudprovider.ModelartsPoolUpdateOption{}, "modelarts-pool-update", "Update Modelarts Pool", func(cli *huawei.SRegion, args *cloudprovider.ModelartsPoolUpdateOption) error {
		res, err := cli.GetClient().Update(args)
		if err != nil {
			return err
		}
		// printList(pools, len(pools), 0, 0, nil)
		log.Infoln("this is res:", res)
		return nil
	})

	shellutils.R(&ModelartsPoolListOption{}, "modelarts-pool-monitor", "Delete Modelarts Pool", func(cli *huawei.SRegion, args *ModelartsPoolListOption) error {
		log.Infof("this is args.poolName", args.PoolId)
		res, err := cli.GetClient().MonitorPool(args.PoolId)
		if err != nil {
			return err
		}
		log.Println("this is res:", res)
		// printList(res.Metrics, len(res.Metrics), 0, 0, nil)
		return nil
	})
}
