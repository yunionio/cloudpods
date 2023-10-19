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
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	apis "yunion.io/x/onecloud/pkg/apis/scheduledtask"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/scheduledtask"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

var (
	R           = shell.R
	printList   = printutils.PrintJSONList
	printObject = printutils.PrintJSONObject
)

func init() {
	type ScheduledTaskListOptions struct {
		options.BaseListOptions

		ScheduledType string `help:"scheduled type" choices:"timing|cycle"`
		ResourceType  string `help:"resource type"`
		Operation     string `help:"operation"`
		UtcOffset     int    `help:"utc offset"`
	}
	R(&ScheduledTaskListOptions{}, "scheduledtask-list", "list Scheduled Task", func(s *mcclient.ClientSession, args *ScheduledTaskListOptions) error {
		params, err := options.ListStructToParams(args)
		tasks, err := modules.ScheduledTask.List(s, params)
		if err != nil {
			return err
		}
		printList(tasks, modules.ScheduledTask.GetColumns(s))
		return nil
	})

	type ScheduledTaskShowOptions struct {
		ID string `help:"ScheduledTask ID or Name"`
	}
	R(&ScheduledTaskShowOptions{}, "scheudled-task-show", "Show Scheduled Task", func(s *mcclient.ClientSession,
		args *ScheduledTaskShowOptions) error {
		params := jsonutils.NewDict()
		params.Set("details", jsonutils.JSONTrue)
		task, err := modules.ScheduledTask.Get(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(task)
		return nil
	})

	type Timer struct {
		TimingExecTime string `help:"Exectime for 'timing' type trigger, format:'2006-01-02 15:04:05'" json:"exec_time"`
	}

	type CycleTimer struct {
		CycleCycleType string `help:"Cycle type for cycle timer" json:"cycle_type" choices:"hour|day|week|month"`
		CycleMinute    int    `help:"Minute of cycle timer" json:"minute"`
		CycleHour      int    `help:"Hour of cycle timer" json:"hour"`
		CycleCycleNum  int    `help:"Cycle count of cycle timer" json:"cycle_num"`
		CycleWeekdays  []int  `help:"Weekdays for cycle timer" json:"weekdays"`
		CycleMonthDays []int  `help:"Month days for cycle timer" json:"month_days"`
		CycleStartTime string `help:"Start time for cycle timer, format:'2006-01-02 15:04:05'" json:"start_time"`
		CycleEndTime   string `help:"End time for cycle timer, format:'2006-01-02 15:04:05'" json:"end_time"`
	}

	type ScheduledTaskCreateOptions struct {
		NAME          string `help:"ScheduledTask Name" json:"name"`
		ScheduledType string `help:"Scheudled Type" choices:"timing|cycle" json:"scheduled_type"`

		Timer
		CycleTimer

		ResourceType string   `help:"resource type"`
		Operation    string   `help:"operation"`
		LabelType    string   `help:"label type"`
		Labels       []string `help:"labels"`
	}
	R(&ScheduledTaskCreateOptions{}, "scheduledtask-create", "Create Scheduled Task", func(s *mcclient.ClientSession, args *ScheduledTaskCreateOptions) error {
		formatStr := "2006-01-02 15:04:05"
		var exectime, starttime, endtime time.Time
		var err error
		if len(args.TimingExecTime) > 0 {
			exectime, err = time.Parse(formatStr, args.TimingExecTime)
			if err != nil {
				return fmt.Errorf("invalid time format for 'exec_time'")
			}
		}
		if len(args.CycleStartTime) > 0 {
			starttime, err = time.Parse(formatStr, args.CycleStartTime)
			if err != nil {
				return fmt.Errorf("invalid time format for 'start_time'")
			}
		}
		if len(args.CycleEndTime) > 0 {
			endtime, err = time.Parse(formatStr, args.CycleEndTime)
			if err != nil {
				return fmt.Errorf("invalid time format for 'end_time'")
			}
		}
		stCreateInput := apis.ScheduledTaskCreateInput{
			ScheduledType: args.ScheduledType,
			Timer: apis.TimerCreateInput{
				ExecTime: exectime,
			},
			CycleTimer: apis.CycleTimerCreateInput{
				CycleType: args.CycleCycleType,
				CycleNum:  args.CycleCycleNum,
				Minute:    args.CycleMinute,
				Hour:      args.CycleHour,
				WeekDays:  args.CycleWeekdays,
				MonthDays: args.CycleMonthDays,
				StartTime: starttime,
				EndTime:   endtime,
			},
			ResourceType: args.ResourceType,
			Operation:    args.Operation,
			LabelType:    args.LabelType,
			Labels:       args.Labels,
		}
		stCreateInput.Name = args.NAME
		ret, err := modules.ScheduledTask.Create(s, jsonutils.Marshal(stCreateInput))
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	type ScheduledTaskEnableOptions struct {
		ID string `help:"ScheduledTask ID or Name"`
	}
	R(&ScheduledTaskEnableOptions{}, "scheduledtask-enable", "Enable ScheduledTask", func(s *mcclient.ClientSession,
		args *ScheduledTaskEnableOptions) error {
		ret, err := modules.ScheduledTask.PerformAction(s, args.ID, "enable", jsonutils.NewDict())
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&ScheduledTaskEnableOptions{}, "scheduledtask-disable", "Disable ScheduledTask",
		func(s *mcclient.ClientSession, args *ScheduledTaskEnableOptions) error {
			ret, err := modules.ScheduledTask.PerformAction(s, args.ID, "disable", jsonutils.NewDict())
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)

	type ScheduledTaskSetLabelsOptions struct {
		ID     string   `help:"ScheduledTask ID or Name"`
		Labels []string `help:"Label"`
	}
	R(&ScheduledTaskSetLabelsOptions{}, "scheduledtask-setlabels", "Trigger ScheduledTask's action",
		func(s *mcclient.ClientSession, args *ScheduledTaskSetLabelsOptions) error {
			params := jsonutils.NewDict()
			params.Set("labels", jsonutils.Marshal(args.Labels))
			ret, err := modules.ScheduledTask.PerformAction(s, args.ID, "set-labels", params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)

	type ScheduledTaskTriggerOptions struct {
		ID string `help:"ScheduledTask ID or Name"`
	}
	R(&ScheduledTaskTriggerOptions{}, "scheduledtask-trigger", "Trigger ScheduledTask",
		func(s *mcclient.ClientSession, args *ScheduledTaskTriggerOptions) error {
			params := jsonutils.NewDict()
			ret, err := modules.ScheduledTask.PerformAction(s, args.ID, "trigger", params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)

	type ScheduledTaskDeleteOptions struct {
		ID string `help:"ScheduledTask ID or Name"`
	}
	R(&ScheduledTaskDeleteOptions{}, "scheduledtask-delete", "Delete ScheduledTask",
		func(s *mcclient.ClientSession, args *ScheduledTaskDeleteOptions) error {
			ret, err := modules.ScheduledTask.Delete(s, args.ID, jsonutils.NewDict())
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)

	type ScheduledTaskAvtivityListOptions struct {
		options.BaseListOptions
		ScheduledTask string `help:"Scheduled Task" json:"scheduled_task"`
	}
	R(&ScheduledTaskAvtivityListOptions{}, "scheduledtask-activity-list", "List Scheduled Task Activity",
		func(s *mcclient.ClientSession, args *ScheduledTaskAvtivityListOptions) error {
			params, err := options.ListStructToParams(args)
			if err != nil {
				return err
			}
			list, err := modules.ScheduledTaskActivity.List(s, params)
			if err != nil {
				return err
			}
			printList(list, modules.ScheduledTaskActivity.GetColumns(s))
			return nil
		},
	)
}
