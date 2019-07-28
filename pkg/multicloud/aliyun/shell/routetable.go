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

	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type RouteTableListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&RouteTableListOptions{}, "routetable-list", "List routetables", func(cli *aliyun.SRegion, args *RouteTableListOptions) error {
		routetables, total, e := cli.GetRouteTables(nil, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(routetables, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type RouteTableShowOptions struct {
		ID string `help:"ID or name of routetable"`
	}
	shellutils.R(&RouteTableShowOptions{}, "routetable-show", "Show routetable", func(cli *aliyun.SRegion, args *RouteTableShowOptions) error {
		routetables, _, e := cli.GetRouteTables([]string{args.ID}, 0, 1)
		if e != nil {
			return e
		}
		if len(routetables) == 0 {
			return fmt.Errorf("No such ID %s", args.ID)
		}
		printObject(routetables[0])
		return nil
	})
}
