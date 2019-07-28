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
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ReservedInstanceListOptions struct {
		Id     []string `help:"IDs of instances to show"`
		Zone   string   `help:"Zone ID"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&ReservedInstanceListOptions{}, "reserved-instance-list", "List intances", func(cli *aws.SRegion, args *ReservedInstanceListOptions) error {
		e := cli.GetReservedInstance()
		if e != nil {
			return e
		}

		e = cli.GetReservedHostOfferings()
		if e != nil {
			return e
		}
		return nil
	})
}
