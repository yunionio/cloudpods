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
	 * 创建报警规则
	 */
	type AlarmDetailOptions struct {
		METRICNAME       string `help:"Metric name of the alarm"`
		ALARMCONDITION   string `help:"Alarm condition template of the alarm"`
		TEMPLATE         string `help:"Alarm condition template of the alarm"`
		ALARMLEVEL       string `help:"Alarm level of the alarm"`
		CONTACT_TYPE     string `help:"Alarm contact type"`
		EXPIRESECONDS    int64  `help:"Expire seconds of the alarm"`
		ESCALATESECONDS  int64  `help:"Escalate seconds of the alarm"`
		ALARMTEMPLATE_ID string `help:"ID of the alarm template"`
		Remark           string `help:"Remark or description of the new alarm"`
	}
	R(&AlarmDetailOptions{}, "alarm-create", "Create a alarm", func(s *mcclient.ClientSession, args *AlarmDetailOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.METRICNAME), "metric_name")
		params.Add(jsonutils.NewString(args.ALARMCONDITION), "alarm_condition")
		params.Add(jsonutils.NewString(args.TEMPLATE), "template")
		params.Add(jsonutils.NewString(args.ALARMLEVEL), "alarm_level")
		params.Add(jsonutils.NewString(args.CONTACT_TYPE), "contact_type")
		params.Add(jsonutils.NewInt(args.EXPIRESECONDS), "expire_seconds")
		params.Add(jsonutils.NewInt(args.ESCALATESECONDS), "escalate_seconds")
		params.Add(jsonutils.NewString(args.ALARMTEMPLATE_ID), "alarm_template_id")
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}
		alarm, err := modules.Alarms.Create(s, params)
		if err != nil {
			return err
		}
		printObject(alarm)
		return nil
	})

	/**
	 * 删除指定ID的报警规则
	 */
	type AlarmDeleteOptions struct {
		ID string `help:"ID of alarm"`
	}
	R(&AlarmDeleteOptions{}, "alarm-delete", "Delete a alarm", func(s *mcclient.ClientSession, args *AlarmDeleteOptions) error {
		alarm, e := modules.Alarms.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(alarm)
		return nil
	})

	/**
	 * 修改指定ID的报警规则
	 */
	type AlarmUpdateOptions struct {
		ID              string `help:"ID of the alarm"`
		Name            string `help:"Name of the new alarm"`
		MetricName      string `help:"Metric name of the alarm"`
		AlarmCondition  string `help:"Alarm condition template of the alarm"`
		Template        string `help:"Alarm condition template of the alarm"`
		AlarmLevel      string `help:"Alarm level of the alarm"`
		ContactType     string `help:"Alarm contact type"`
		ExpireSeconds   int64  `help:"Expire seconds of the alarm"`
		EscalateSeconds int64  `help:"Escalate seconds of the alarm"`
		Remark          string `help:"Remark or description of the new alarm"`
	}
	R(&AlarmUpdateOptions{}, "alarm-update", "Update a alarm", func(s *mcclient.ClientSession, args *AlarmUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.MetricName) > 0 {
			params.Add(jsonutils.NewString(args.MetricName), "metric_name")
		}
		if len(args.AlarmCondition) > 0 {
			params.Add(jsonutils.NewString(args.AlarmCondition), "alarm_condition")
		}
		if len(args.Template) > 0 {
			params.Add(jsonutils.NewString(args.Template), "template")
		}
		if len(args.AlarmLevel) > 0 {
			params.Add(jsonutils.NewString(args.AlarmLevel), "alarm_level")
		}
		if len(args.ContactType) > 0 {
			params.Add(jsonutils.NewString(args.ContactType), "contact_type")
		}
		if args.ExpireSeconds > 0 {
			params.Add(jsonutils.NewInt(args.ExpireSeconds), "expire_seconds")
		}
		if args.EscalateSeconds > 0 {
			params.Add(jsonutils.NewInt(args.EscalateSeconds), "escalate_seconds")
		}
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}

		alarm, err := modules.Labels.Patch(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(alarm)
		return nil
	})

	/**
	 * 列出报警规则
	 */
	type AlarmListOptions struct {
		options.BaseListOptions
	}
	R(&AlarmListOptions{}, "alarm-list", "List all alarms", func(s *mcclient.ClientSession, args *AlarmListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Alarms.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Alarms.GetColumns(s))
		return nil
	})

	/**
	 * 根据ID查询报警规则
	 */
	type AlarmShowOptions struct {
		ID string `help:"ID of the alarm to show"`
	}
	R(&AlarmShowOptions{}, "alarm-show", "Show alarm details", func(s *mcclient.ClientSession, args *AlarmShowOptions) error {
		result, err := modules.Alarms.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
