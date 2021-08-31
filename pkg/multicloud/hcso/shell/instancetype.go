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
	huawei "yunion.io/x/onecloud/pkg/multicloud/hcso"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceMatchOptions struct {
		CPU  int    `help:"CPU count"`
		MEM  int    `help:"Memory in MB"`
		Zone string `help:"Test in zone"`
	}
	shellutils.R(&InstanceMatchOptions{}, "instance-type-select", "Select matching instance types", func(cli *huawei.SRegion, args *InstanceMatchOptions) error {
		instanceTypes, e := cli.GetMatchInstanceTypes(args.CPU, args.MEM, args.Zone)
		if e != nil {
			return e
		}
		printList(instanceTypes, 0, 0, 0, []string{})
		return nil
	})
}
