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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type Timer struct {
	TimingExecTime string `help:"Exectime for 'timing' type trigger, format:'2006-01-02 15:04:05'" json:"exec_time"`
}

type CycleTimer struct {
	CycleCycleType string `help:"Cycle type for cycle timer" json:"cycle_type" choices:"day|week|month"`
	CycleMinute    int    `help:"Minute of cycle timer" json:"minute"`
	CycleHour      int    `help:"Hour of cycle timer" json:"hour"`
	CycleWeekdays  []int  `help:"Weekdays for cycle timer" json:"weekdays"`
	CycleMonthDays []int  `help:"Month days for cycle timer" json:"month_days"`
	CycleStartTime string `help:"Start time for cycle timer, format:'2006-01-02 15:04:05'" json:"start_time"`
	CycleEndTime   string `help:"End time for cycle timer, format:'2006-01-02 15:04:05'" json:"end_time"`
}

func init() {
	type ScalingPolicyListOptions struct {
		options.BaseListOptions
		ScalingGroup string `help:"ScalingGroup ID or Name"`
		TriggerType  string `help:"Trigger type" choices:"alarm|timing|cycle"`
	}
	R(&ScalingPolicyListOptions{}, "scaling-policy-list", "List Scaling Policy", func(s *mcclient.ClientSession,
		args *ScalingPolicyListOptions) error {
		params, err := options.ListStructToParams(args)
		policies, err := modules.ScalingPolicy.List(s, params)
		if err != nil {
			return err
		}
		printList(policies, modules.ScalingPolicy.GetColumns(s))
		return nil
	})

	type ScalingPolicyShowOptions struct {
		ID string `help:"ScalingPolicy ID or Name"`
	}
	R(&ScalingPolicyShowOptions{}, "scaling-policy-show", "Show Scaling Policy", func(s *mcclient.ClientSession,
		args *ScalingPolicyShowOptions) error {
		params := jsonutils.NewDict()
		params.Set("details", jsonutils.JSONTrue)
		policy, err := modules.ScalingPolicy.Get(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(policy)
		return nil
	})

	type ScalingAlarm struct {
		AlarmCumulate  int     `help:"Cumulate times alarm will trigger, for 'alarm' trigger" json:"cumulate"`
		AlarmCycle     int     `help:"Monitoring cycle for indicators, for 'alarm' trigger" json:"cycle"`
		AlarmIndicator string  `help:"Indicator for 'alarm' trigger" json:"indicator"`
		AlarmWrapper   string  `help:"Wrapper for Indicators" choices:"max|min|average" json:"wrapper"`
		AlarmOperator  string  `help:"Operator between Indicator and Operator" json:"operator"`
		AlarmValue     float64 `help:"Value of Indicator" json:"value"`
	}

	type ScalingPolicyCreateOptions struct {
		NAME         string `help:"ScalingPolicy Name" json:"name"`
		ScalingGroup string `help:"ScalingGroup ID or Name" json:"scaling_group"`
		TriggerType  string `help:"Trigger type" choices:"alarm|timing|cycle" json:"trigger_type"`

		Timer
		CycleTimer
		ScalingAlarm

		Action      string `help:"Action for scaling policy" choices:"add|remove|set" json:"action"`
		Number      int    `help:"Instance number for action" json:"number"`
		Unit        string `help:"Unit for Number" choices:"s|%" json:"unit"`
		CoolingTime int    `help:"Cooling time, unit: s" json:"cooling_time"`
	}

	R(&ScalingPolicyCreateOptions{}, "scaling-policy-create", "Create Scaling Policy",
		func(s *mcclient.ClientSession, args *ScalingPolicyCreateOptions) error {
			formatStr := "2006-01-02 15:04:05"
			timingExecTime, err := time.Parse(formatStr, args.TimingExecTime)
			if err != nil {
				return fmt.Errorf("invalid time format for 'exec_time'")
			}
			cycleStarTime, err := time.Parse(formatStr, args.CycleStartTime)
			if err != nil {
				return fmt.Errorf("invalid time format for 'start_time'")
			}
			cycleEndTime, err := time.Parse(formatStr, args.CycleEndTime)
			if err != nil {
				return fmt.Errorf("invalid time format for 'end_time'")
			}
			spCreateInput := api.ScalingPolicyCreateInput{
				ScalingGroup: args.ScalingGroup,
				TriggerType:  args.TriggerType,
				Timer: api.TimerCreateInput{
					ExecTime: timingExecTime,
				},
				CycleTimer: api.CycleTimerCreateInput{
					CycleType: args.CycleCycleType,
					Minute:    args.CycleMinute,
					Hour:      args.CycleHour,
					WeekDays:  args.CycleWeekdays,
					MonthDays: args.CycleMonthDays,
					StartTime: cycleStarTime,
					EndTime:   cycleEndTime,
				},
				Alarm: api.ScalingAlarmCreateInput{
					Cumulate:  args.AlarmCumulate,
					Cycle:     args.AlarmCycle,
					Indicator: args.AlarmIndicator,
					Wrapper:   args.AlarmWrapper,
					Operator:  args.AlarmOperator,
					Value:     args.AlarmValue,
				},
				Action:      args.Action,
				Number:      args.Number,
				Unit:        args.Unit,
				CoolingTime: args.CoolingTime,
			}
			spCreateInput.Name = args.NAME
			ret, err := modules.ScalingPolicy.Create(s, jsonutils.Marshal(spCreateInput))
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)

	type ScalingPolicyEnableOptions struct {
		ID string `help:"ScalingPolicy ID or Name"`
	}
	R(&ScalingPolicyEnableOptions{}, "scaling-policy-enable", "Enable ScalingPolicy", func(s *mcclient.ClientSession,
		args *ScalingPolicyEnableOptions) error {
		ret, err := modules.ScalingPolicy.PerformAction(s, args.ID, "enable", jsonutils.NewDict())
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&ScalingPolicyEnableOptions{}, "scaling-policy-disable", "Disable ScalingPolicy",
		func(s *mcclient.ClientSession, args *ScalingPolicyEnableOptions) error {
			ret, err := modules.ScalingPolicy.PerformAction(s, args.ID, "disable", jsonutils.NewDict())
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)

	type ScalingPolicyTriggerOptions struct {
		ID string `help:"ScalingPolicy ID or Name"`
	}
	R(&ScalingPolicyTriggerOptions{}, "scaling-policy-trigger", "Trigger ScalingPolicy's action",
		func(s *mcclient.ClientSession, args *ScalingPolicyTriggerOptions) error {
			ret, err := modules.ScalingPolicy.PerformAction(s, args.ID, "trigger", jsonutils.NewDict())
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)

	type ScalingPolicyDeleteOptions struct {
		ID string `help:"ScalingPolicy ID or Name"`
	}
	R(&ScalingPolicyDeleteOptions{}, "scaling-policy-delete", "Delete SclaingPolicy",
		func(s *mcclient.ClientSession, args *ScalingPolicyDeleteOptions) error {
			ret, err := modules.ScalingPolicy.Delete(s, args.ID, jsonutils.NewDict())
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		},
	)
}
