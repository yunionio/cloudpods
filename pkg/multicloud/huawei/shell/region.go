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
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type RegionListOptions struct {
	}
	shellutils.R(&RegionListOptions{}, "region-list", "List regions", func(cli *huawei.SRegion, args *RegionListOptions) error {
		regions := cli.GetClient().GetRegions()
		printList(regions, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&RegionListOptions{}, "project-list", "List projects", func(cli *huawei.SRegion, args *RegionListOptions) error {
		projects, err := cli.GetClient().GetProjects()
		if err != nil {
			return err
		}
		printList(projects, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&RegionListOptions{}, "domain-list", "List domains", func(cli *huawei.SRegion, args *RegionListOptions) error {
		domains, err := cli.GetClient().GetDomains()
		if err != nil {
			return err
		}
		printList(domains, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&RegionListOptions{}, "capabilities", "Get capabilities", func(cli *huawei.SRegion, args *RegionListOptions) error {
		capabilities := cli.GetClient().GetCapabilities()
		fmt.Println(capabilities)
		return nil
	})
}
