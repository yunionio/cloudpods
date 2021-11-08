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
	type Top5Options struct {
		NODE_LABELS string `help:"Service tree tree-node labels"`
	}
	R(&Top5Options{}, "performance-top5", "Show performance top5", func(s *mcclient.ClientSession, args *Top5Options) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NODE_LABELS), "node_labels")

		result, err := modules.Performances.GetTop5(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
