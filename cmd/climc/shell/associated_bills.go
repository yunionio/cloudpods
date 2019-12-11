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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type AssociatedBillListOptions struct {
		options.BaseListOptions
		StartDay int `help:"time range of the associated bills, example: 20060102" required:"true"`
		EndDay   int `help:"time range of the associated bills, example: 20060102" required:"true"`

		ResourceId string `help:"resource id"`
		Brand      string `help:"brand"`
	}
	R(&AssociatedBillListOptions{}, "associated-bill-list", "List associated bills",
		func(s *mcclient.ClientSession, args *AssociatedBillListOptions) error {
			params, err := options.ListStructToParams(args)
			if err != nil {
				return err
			}
			result, err := modules.AssociatedBills.List(s, params)
			if err != nil {
				return err
			}
			printList(result, modules.AssociatedBills.GetColumns(s))
			return nil
		})
}
