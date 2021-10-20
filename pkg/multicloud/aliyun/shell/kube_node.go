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
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type KubeNodeListOptions struct {
		CLUSTER    string
		PageSize   int
		PageNumber int
	}
	shellutils.R(&KubeNodeListOptions{}, "kube-node-list", "List kube nodes", func(cli *aliyun.SRegion, args *KubeNodeListOptions) error {
		nodes, _, err := cli.GetKubeNodes(args.CLUSTER, args.PageSize, args.PageNumber)
		if err != nil {
			return err
		}
		printList(nodes, 0, 0, 0, []string{})
		return nil
	})

}
