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
	"fmt"

	monitorapi "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	aN := cmdN("alert")
	R(&options.AlertListOptions{}, aN("list"), "List all alerts",
		func(s *mcclient.ClientSession, args *options.AlertListOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.Alerts.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, monitor.Alerts.GetColumns(s))
			return nil
		})

	R(&options.AlertCreateOptions{}, aN("create"), "Create alert rule",
		func(s *mcclient.ClientSession, args *options.AlertCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.Alerts.DoCreate(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.AlertShowOptions{}, aN("show"), "Show details of a alert rule",
		func(s *mcclient.ClientSession, args *options.AlertShowOptions) error {
			ret, err := monitor.Alerts.Get(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.AlertUpdateOptions{}, aN("update"), "Update a alert rule",
		func(s *mcclient.ClientSession, args *options.AlertUpdateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.Alerts.Update(s, args.ID, params.JSON(params))
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.AlertDeleteOptions{}, aN("delete"), "Delete alerts",
		func(s *mcclient.ClientSession, args *options.AlertDeleteOptions) error {
			ret := monitor.Alerts.BatchDelete(s, args.ID, nil)
			printBatchResults(ret, monitor.Alerts.GetColumns(s))
			return nil
		})

	R(&options.AlertTestRunOptions{}, aN("test-run"), "Test run alert",
		func(s *mcclient.ClientSession, args *options.AlertTestRunOptions) error {
			data := new(monitorapi.AlertTestRunInput)
			if args.Debug {
				data.IsDebug = true
			}
			ret, err := monitor.Alerts.DoTestRun(s, args.ID, data)
			if err != nil {
				return err
			}
			fmt.Println(ret.JSON(ret).YAMLString())
			return nil
		})
}
