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

package meter

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type BillTasksCreateOptions struct {
		// options.BaseListOptions
		CloudaccountId string `help:"cloudaccount Id" required:"true"`
		StartDay       int    `help:"start day of billing cycle, example: 20060102"`
		EndDay         int    `help:"end day of billing cycle, example: 20060102"`
		TaskType       string `help:"task type" choices:"bill_remove|bill_pulling"`
	}
	R(&BillTasksCreateOptions{}, "bill-tasks-create", "create bill task",
		func(s *mcclient.ClientSession, args *BillTasksCreateOptions) error {
			params := jsonutils.Marshal(args)
			result, err := modules.BillTasks.Create(s, params)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		})
}
