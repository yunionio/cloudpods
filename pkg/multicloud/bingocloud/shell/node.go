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
	"yunion.io/x/onecloud/pkg/multicloud/bingocloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NodeListOptions struct {
		ClusterId string
		NodeId    string
	}
	shellutils.R(&NodeListOptions{}, "node-list", "List nodes", func(cli *bingocloud.SRegion, args *NodeListOptions) error {
		nodes, err := cli.GetNodes(args.ClusterId, args.NodeId)
		if err != nil {
			return err
		}
		printList(nodes, 0, 0, 0, nil)
		return nil
	})
}
