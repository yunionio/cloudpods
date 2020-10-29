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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type RegionListOptions struct {
	}
	shellutils.R(&RegionListOptions{}, "region-list", "List regions", func(cli *azure.SRegion, args *RegionListOptions) error {
		regions := cli.GetClient().GetRegions()
		printList(regions, 0, 0, 0, nil)
		return nil
	})

	type ResourceIdOptions struct {
		ID string `help:"resource id"`
	}
	shellutils.R(&ResourceIdOptions{}, "delete", "delete resource", func(cli *azure.SRegion, args *ResourceIdOptions) error {
		return cli.Delete(args.ID)
	})

	shellutils.R(&ResourceIdOptions{}, "show", "Show resource", func(cli *azure.SRegion, args *ResourceIdOptions) error {
		ret, err := cli.Show(args.ID)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

}
