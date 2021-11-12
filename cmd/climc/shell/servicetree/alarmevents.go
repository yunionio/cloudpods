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
	 * 列出报警事件
	 */
	R(&options.AlarmEventListOptions{}, "alarmevent-list", "List all alarm's event", func(s *mcclient.ClientSession, opts *options.AlarmEventListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.AlarmEvents.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.AlarmEvents.GetColumns(s))
		return nil
	})

	/*
	 * 修改报警事件ACK确认状态
	 */
	type AlarmEventUpdateOptions struct {
		ID     string `help:"ID of the alarm event"`
		STATUS int64  `help:"Alarm event ack status" choices:"0|1"`
	}
	R(&AlarmEventUpdateOptions{}, "alarmevent-ack-update", "Update ack status of alarm event", func(s *mcclient.ClientSession, args *AlarmEventUpdateOptions) error {
		arr := jsonutils.NewArray()
		tmpObj := jsonutils.NewDict()
		tmpObj.Add(jsonutils.NewString(args.ID), "id")
		tmpObj.Add(jsonutils.NewInt(args.STATUS), "ack_status")
		arr.Add(tmpObj)

		params := jsonutils.NewDict()
		params.Add(arr, "alarm_events")

		alarmevent, err := modules.AlarmEvents.DoBatchUpdate(s, params)
		if err != nil {
			return err
		}
		printObject(alarmevent)
		return nil
	})

}
