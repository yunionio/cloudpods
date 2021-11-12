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

package servicetree

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/servicetree"
)

func init() {

	/**
	 * move host to one tree-node
	 */
	type ProjectHostCreateOptions struct {
		LABELS    string   `help:"Labels for tree-node(split by comma)"`
		HOST_NAME []string `help:"Host names to move to"`
	}
	R(&ProjectHostCreateOptions{}, "projecthost-create", "move host to tree-node", func(s *mcclient.ClientSession, args *ProjectHostCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")
		arr := jsonutils.NewArray()
		if len(args.HOST_NAME) > 0 {
			for _, f := range args.HOST_NAME {
				tmpObj := jsonutils.NewDict()
				tmpObj.Add(jsonutils.NewString(f), "host_name")
				arr.Add(tmpObj)
			}
		}
		params.Add(arr, "service_hosts")

		rst, err := modules.ProjectHosts.Create(s, params)

		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})
}
