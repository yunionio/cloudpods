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
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 列出所有监控数据源
	 */
	type MonitorInputsListOptions struct {
		options.BaseListOptions
	}
	R(&MonitorInputsListOptions{}, "monitorinputs-list", "List all monitor-inputs", func(s *mcclient.ClientSession, args *MonitorInputsListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.MonitorInputs.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.MonitorInputs.GetColumns(s))
		return nil
	})

	/**
	 * 查看监控数据源下的监控项
	 */
	type MonitorInputsShowOptions struct {
		options.BaseListOptions
		ID string `help:"The ID of the monitor-input"`
	}
	R(&MonitorInputsShowOptions{}, "monitorinputs-metrics-list", "List all metrics for the monitor-inputs", func(s *mcclient.ClientSession, args *MonitorInputsShowOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.MonitorInputs.GetSpecific(s, args.ID, "metrics", params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

}
