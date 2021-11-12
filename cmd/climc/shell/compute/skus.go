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
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.ServerSkus).WithKeyword("server-sku")
	cmd.List(&options.ServerSkusListOptions{})
	cmd.Show(&options.ServerSkusIdOptions{})
	cmd.Delete(&options.ServerSkusIdOptions{})
	cmd.Perform("enable", &options.ServerSkusIdOptions{})
	cmd.Perform("disable", &options.ServerSkusIdOptions{})
	cmd.Create(&options.ServerSkusCreateOptions{})
	cmd.Update(&options.ServerSkusUpdateOptions{})
	cmd.ClassShow(&options.ServerSkusListOptions{})
	cmd.PerformClass("sync-skus", &options.SkuSyncOptions{})

	R(&options.SkuTaskQueryOptions{}, "server-sku-sync-task-show", "Show details of skus sync tasks", func(s *mcclient.ClientSession, args *options.SkuTaskQueryOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}

		result, err := modules.ServerSkus.Get(s, "sync-tasks", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
