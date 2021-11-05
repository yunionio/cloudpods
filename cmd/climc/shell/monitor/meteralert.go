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

package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 创建一条报警规则
	 */
	type MeterAlertCreateOptions struct {
		Type       string  `help:"Alert rule type" required:"true" choices:"balance|resFee|monthFee"`
		Threshold  float64 `help:"Threshold value of the metric" required:"true"`
		Comparator string  `help:"Comparison operator for join expressions" required:"true" choices:">|<|>=|<=|=|!="`
		Recipients string  `help:"Comma separated recipient ID" required:"true"`
		Level      string  `help:"Alert level" required:"true" choices:"normal|important|fatal"`
		Channel    string  `help:"Ways to send an alarm" required:"true" choices:"email|mobile"`
		Provider   string  `help:"Name of the cloud platform"`
		AccountId  string  `help:"ID of the cloud platform"`
		ProjectId  string  `help:"ID of the project" required:"true"`
	}
	R(&MeterAlertCreateOptions{}, "meteralert-create", "Create a meter alert rule", func(s *mcclient.ClientSession, args *MeterAlertCreateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}

		rst, err := modules.MeterAlert.Create(s, params)
		if err != nil {
			return err
		}

		printObject(rst)
		return nil
	})

	/**
	 * 删除指定ID的报警规则
	 */
	type MeterAlertDeleteOptions struct {
		ID string `help:"ID of alarm"`
	}
	R(&MeterAlertDeleteOptions{}, "meteralert-delete", "Delete a meter alert", func(s *mcclient.ClientSession, args *MeterAlertDeleteOptions) error {
		alarm, e := modules.MeterAlert.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(alarm)
		return nil
	})

	/**
	 * 修改指定ID的报警规则状态
	 */
	type MeterAlertUpdateOptions struct {
		ID     string `help:"ID of the meter alert"`
		STATUS string `help:"Name of the new alarm" choices:"Enabled|Disabled"`
	}
	R(&MeterAlertUpdateOptions{}, "meteralert-change-status", "Change status of a meter alert", func(s *mcclient.ClientSession, args *MeterAlertUpdateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.STATUS), "status")

		alarm, err := modules.MeterAlert.Patch(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(alarm)
		return nil
	})

	/**
	 * 列出报警规则
	 */
	type MeterAlertListOptions struct {
		Type          string `help:"Alarm rule type" choices:"balance|resFee|monthFee"`
		CloudProvider string `help:"Name of cloud provider, case sensitive"`
		AccountId     string `help:"Cloud account ID"`
		ProjectId     string `help:"ID of the project" required:"true"`
		options.BaseListOptions
	}
	R(&MeterAlertListOptions{}, "meteralert-list", "List meter alert", func(s *mcclient.ClientSession, args *MeterAlertListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}

		result, err := modules.MeterAlert.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.MeterAlert.GetColumns(s))
		return nil
	})
}
