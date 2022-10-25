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
	type InstanceMatchOptions struct {
		Zone string `help:"Test in zone"`
	}
	shellutils.R(&InstanceMatchOptions{}, "instance-type-list", "Select matching instance types", func(cli *hcs.SRegion, args *InstanceMatchOptions) error {
		instanceTypes, e := cli.GetchInstanceTypes(args.Zone)
		if e != nil {
			return e
		}
		printList(instanceTypes, 0, 0, 0, []string{})
		return nil
	})

	type InstanceTypeIdOptions struct {
		ID string
	}

	shellutils.R(&InstanceTypeIdOptions{}, "instance-type-show", "show instance type", func(cli *hcs.SRegion, args *InstanceTypeIdOptions) error {
		ret, e := cli.GetchInstanceType(args.ID)
		if e != nil {
			return e
		}
		printObject(ret)
		return nil
	})

}
