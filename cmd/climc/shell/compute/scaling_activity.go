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

package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ScalingActivityListOptions struct {
		options.BaseListOptions
	}
	R(&ScalingActivityListOptions{}, "scaling-activity-list", "List Scaling Activity",
		func(s *mcclient.ClientSession, args *ScalingActivityListOptions) error {
			params, err := options.ListStructToParams(args)
			if err != nil {
				return err
			}
			list, err := modules.ScalingActivity.List(s, params)
			if err != nil {
				return err
			}
			printList(list, modules.ScalingActivity.GetColumns(s))
			return nil
		},
	)
}
