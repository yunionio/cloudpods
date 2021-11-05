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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/servicetree"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 创建报警日志
	 */
	type AlarmLogDetailOptions struct {
		NODE_NAME      string `help:"Name of the host"`
		Lables         string `help:"Labels of the service tree node"`
		Metric_Name    string `help:"Metric name of the alarm"`
		Start_Time     string `help:"Alarm start time"`
		This_Time      string `help:"Alarm this time"`
		Alarm_Ways     string `help:"Alarm ways"`
		Alarm_Level    string `help:"Alarm level"`
		Alarm_Status   string `help:"Alarm status"`
		Receive_Person string `help:"Alarm receive person"`
		Reason         string `help:"Alarm reason, aka alarm rule content"`
		Alarm_Id       string `help:"Alarm ID"`
		Host_Id        string `help:"ID of the host"`
	}
	R(&AlarmLogDetailOptions{}, "alarmlog-create", "Create a alarm log", func(s *mcclient.ClientSession, args *AlarmLogDetailOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NODE_NAME), "node_name")
		if len(args.Lables) > 0 {
			params.Add(jsonutils.NewString(args.Lables), "labels")
		}
		if len(args.Metric_Name) > 0 {
			params.Add(jsonutils.NewString(args.Metric_Name), "metric_name")
		}
		if len(args.Start_Time) > 0 {
			params.Add(jsonutils.NewString(args.Start_Time), "start_time")
		}
		if len(args.This_Time) > 0 {
			params.Add(jsonutils.NewString(args.This_Time), "this_time")
		}
		if len(args.Alarm_Ways) > 0 {
			params.Add(jsonutils.NewString(args.Alarm_Ways), "alarm_ways")
		}
		if len(args.Alarm_Level) > 0 {
			params.Add(jsonutils.NewString(args.Alarm_Level), "alarm_level")
		}
		if len(args.Alarm_Status) > 0 {
			params.Add(jsonutils.NewString(args.Alarm_Status), "alarm_status")
		}
		if len(args.Receive_Person) > 0 {
			params.Add(jsonutils.NewString(args.Receive_Person), "receive_person")
		}
		if len(args.Reason) > 0 {
			params.Add(jsonutils.NewString(args.Reason), "reason")
		}
		if len(args.Alarm_Id) > 0 {
			params.Add(jsonutils.NewString(args.Alarm_Id), "alarm_id")
		}
		if len(args.Host_Id) > 0 {
			params.Add(jsonutils.NewString(args.Host_Id), "host_id")
		}
		alarmlog, err := modules.AlarmLogs.Create(s, params)
		if err != nil {
			return err
		}
		printObject(alarmlog)
		return nil
	})

	/**
	 * 列出报警日志
	 */
	type AlarmLogListOptions struct {
		options.BaseListOptions
		Host           string   `help:"Name of the host"`
		Id             int64    `help:"ID of tree node"`
		Label          []string `help:"Labels to this tree node"`
		Metric         string   `help:"Metric name"`
		StartTimeSince string   `help:"Start time since the alarm event"`
		StartTimeUntil string   `help:"Start time until the alarm event"`
		ThisTimeSince  string   `help:"This time since the alarm event"`
		ThisTimeUntil  string   `help:"This time until the alarm event"`
	}
	R(&AlarmLogListOptions{}, "alarmlog-list", "List all alarm's event", func(s *mcclient.ClientSession, args *AlarmLogListOptions) error {
		params, err := args.BaseListOptions.Params()
		if err != nil {
			return err
		}
		if len(args.Host) > 0 {
			params.Add(jsonutils.NewString(args.Host), "host")
		}
		if len(args.Label) > 0 {
			names := []string{"corp", "owt", "pdl", "srv", "env"}
			segs := make([]string, len(args.Label))

			for i := 0; i < len(args.Label); i += 1 {
				sublabel := args.Label[:i+1]
				pid, _ := modules.TreeNodes.GetNodeIDByLabels(s, sublabel)
				if pid < 0 {
					return fmt.Errorf("Invalid node data")
				}
				segs[i] = fmt.Sprintf("%s=%d", names[i], pid)
			}

			node_labels := strings.Join(segs, ",")

			params.Add(jsonutils.NewString(node_labels), "node_labels")
		}
		if len(args.Metric) > 0 {
			params.Add(jsonutils.NewString(args.Metric), "metric")
		}
		if len(args.StartTimeSince) > 0 {
			params.Add(jsonutils.NewString(args.StartTimeSince), "start_time_since")
		}
		if len(args.StartTimeUntil) > 0 {
			params.Add(jsonutils.NewString(args.StartTimeUntil), "start_time_until")
		}
		if len(args.ThisTimeSince) > 0 {
			params.Add(jsonutils.NewString(args.ThisTimeSince), "this_time_since")
		}
		if len(args.ThisTimeUntil) > 0 {
			params.Add(jsonutils.NewString(args.ThisTimeUntil), "this_time_until")
		}

		result, err := modules.AlarmLogs.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.AlarmLogs.GetColumns(s))
		return nil
	})

}
