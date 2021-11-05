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
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/servicetree"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 创建报警模板
	 */
	type AlarmTemplateOptions struct {
		NAME        string `help:"Name of the new alarm-template"`
		BELONGTO    string `help:"Detect the service-tree-node' labels which alarm-template belong to"`
		Description string `help:"Description of the alarm-template"`
	}
	R(&AlarmTemplateOptions{}, "alarmtemplate-create", "Create a alarm-template", func(s *mcclient.ClientSession, args *AlarmTemplateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "alarm_template_name")
		params.Add(jsonutils.NewString(args.BELONGTO), "belongto")
		if len(args.Description) > 0 {
			params.Add(jsonutils.NewString(args.Description), "alarm_template_desc")
		}
		alarmTemplate, err := modules.AlarmTemplates.Create(s, params)
		if err != nil {
			return err
		}
		printObject(alarmTemplate)
		return nil
	})

	/**
	 * 删除指定ID的报警模板
	 */
	type AlarmTemplateDeleteOptions struct {
		ID string `help:"ID of alarm-template"`
	}
	R(&AlarmTemplateDeleteOptions{}, "alarmtemplate-delete", "Delete a alarm-template", func(s *mcclient.ClientSession, args *AlarmTemplateDeleteOptions) error {
		alarmTemplate, e := modules.AlarmTemplates.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(alarmTemplate)
		return nil
	})

	/**
	 * 修改指定ID的报警模板
	 */
	type AlarmTemplateUpdateOptions struct {
		ID          string `help:"ID of the alarm-template"`
		Name        string `help:"New name of the new alarm-template"`
		Description string `help:"New description of the alarm-template"`
	}
	R(&AlarmTemplateUpdateOptions{}, "alarmtemplate-update", "Update a alarm-template", func(s *mcclient.ClientSession, args *AlarmTemplateUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "alarm_template_name")
		}
		if len(args.Description) > 0 {
			params.Add(jsonutils.NewString(args.Description), "alarm_template_desc")
		}

		alarmTemplate, err := modules.AlarmTemplates.Put(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(alarmTemplate)
		return nil
	})

	/**
	 * 列出报警模板
	 */
	type AlarmTemplateListOptions struct {
		options.BaseListOptions
		LIST_TYPE string `help:"Type of list: avaliable|applied|created "`
		LABELS    string `help:"Labels for node(split by comma)"`
	}
	R(&AlarmTemplateListOptions{}, "alarmtemplate-list", "List all alarm templates ", func(s *mcclient.ClientSession, args *AlarmTemplateListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		params.Add(jsonutils.NewString(args.LIST_TYPE), "type")
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")

		result, err := modules.AlarmTemplates.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.AlarmTemplates.GetColumns(s))
		return nil
	})

	/**
	 * 根据ID查询报警模板（查看报警模板详情）
	 */
	type AlarmTemplateShowOptions struct {
		ID string `help:"ID of the alarm-template to show"`
	}
	R(&AlarmTemplateShowOptions{}, "alarmtemplate-show", "Show alarm template details", func(s *mcclient.ClientSession, args *AlarmTemplateShowOptions) error {
		result, err := modules.AlarmTemplates.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 查看报警模板下的报警项
	 */
	type AlarmTemplateBaseOptions struct {
		ID string `help:"ID of the alarm template"`
	}
	R(&AlarmTemplateBaseOptions{}, "alarmtemplate-alarm-list", "List alarms of alarm template", func(s *mcclient.ClientSession, args *AlarmTemplateBaseOptions) error {
		result, err := modules.AlarmTemplateAlarms.ListDescendent(s, args.ID, nil)
		if err != nil {
			return err
		}
		printList(result, modules.Alarms.GetColumns(s))
		return nil
	})

	/**
	 * 向指定的报警模板中添加报警项（添加报警项）
	 */
	type AlarmTemplateAlarmOptions struct {
		ALARMTEMPLATE_ID string `help:"ID of alarm-template"`
		ALARM_ID         string `help:"ID of alarm"`
	}
	R(&AlarmTemplateAlarmOptions{}, "add-alarm-to-template", "Add alarm to alarm-template", func(s *mcclient.ClientSession, args *AlarmTemplateAlarmOptions) error {
		_, err := modules.AlarmTemplateAlarms.Attach(s, args.ALARMTEMPLATE_ID, args.ALARM_ID, nil)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 从指定的报警模板中删除报警项（删除报警项）
	 */
	R(&AlarmTemplateAlarmOptions{}, "remove-alarm-from-template", "Remove alarm from alarm-template", func(s *mcclient.ClientSession, args *AlarmTemplateAlarmOptions) error {
		_, err := modules.Alarms.DeleteInContext(s, args.ALARM_ID, nil, &modules.AlarmTemplates, args.ALARMTEMPLATE_ID)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 给服务树节点增加报警模板
	 */
	type AlarmTemplateNodeAddOptions struct {
		ID        string   `help:"ID of alarm-template"`
		Labels    []string `help:"Labels for node" nargs:"+"`
		Receiver1 string   `help:"Primary alarm receiver" nargs:"+"`
		Receiver2 string   `help:"Secondary alarm receiver"`
	}
	R(&AlarmTemplateNodeAddOptions{}, "bind-template-to-node", "Bind alarm-template to service-tree node", func(s *mcclient.ClientSession, args *AlarmTemplateNodeAddOptions) error {
		labels := jsonutils.NewDict()
		for _, f := range args.Labels {
			parts := strings.Split(f, "=")
			labels.Add(jsonutils.NewString(parts[1]), parts[0])
		}

		params := jsonutils.NewDict()
		params.Add(labels, "labels")
		params.Add(jsonutils.NewString(args.Receiver1), "tier1_receivers")
		params.Add(jsonutils.NewString(args.Receiver2), "tier2_receivers")

		_, err := modules.AlarmTemplates.PerformAction(s, args.ID, "add-labels", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 将报警模板从服务数节点移除
	 */
	type AlarmTemplateRemoveFromNodeOptions struct {
		ID     string   `help:"ID of the alarm-template"`
		Labels []string `help:"Node labels" nargs:"+"`
	}
	R(&AlarmTemplateRemoveFromNodeOptions{}, "unbind-template-from-node", "Unbind alarm-template from service-tree node", func(s *mcclient.ClientSession, args *AlarmTemplateRemoveFromNodeOptions) error {
		labels := jsonutils.NewDict()
		for _, f := range args.Labels {
			parts := strings.Split(f, "=")
			labels.Add(jsonutils.NewString(parts[1]), parts[0])
		}

		params := jsonutils.NewDict()
		params.Add(labels, "labels")

		_, err := modules.AlarmTemplates.PerformAction(s, args.ID, "remove-labels", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 给服务树编辑模板详情
	 */
	type AlarmTemplateNodeUpdateOptions struct {
		ID        string   `help:"ID of alarm-template"`
		Labels    []string `help:"Labels for node" nargs:"+"`
		Receiver1 string   `help:"Primary alarm receiver"`
		Receiver2 string   `help:"Secondary alarm receiver"`
	}
	R(&AlarmTemplateNodeUpdateOptions{}, "update-template-for-node", "Update alarm-template for service-tree node", func(s *mcclient.ClientSession, args *AlarmTemplateNodeUpdateOptions) error {
		labels := jsonutils.NewDict()
		for _, f := range args.Labels {
			parts := strings.Split(f, "=")
			labels.Add(jsonutils.NewString(parts[1]), parts[0])
		}

		params := jsonutils.NewDict()
		params.Add(labels, "labels")
		params.Add(jsonutils.NewString(args.Receiver1), "tier1_receivers")
		params.Add(jsonutils.NewString(args.Receiver2), "tier2_receivers")

		_, err := modules.AlarmTemplates.PerformAction(s, args.ID, "update-labels", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 查看此模板有哪些label正在使用
	 */
	type AlarmTemplateLabelListOptions struct {
		ID string `help:"ID of alarm-template"`
	}
	R(&AlarmTemplateLabelListOptions{}, "labels-for-alarmtemplate", "List all the labels for the alarm-templates", func(s *mcclient.ClientSession, args *AlarmTemplateLabelListOptions) error {
		result, err := modules.AlarmTemplates.GetSpecific(s, args.ID, "labels", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	/**
	 * 为树务树节点添加报警模板
	 */
	type AlarmTemplateAddToNodeOptions struct {
		ID     string `help:"ID of alarm-template"`
		LABELS string `help:"Labels for node"`
		Status int64  `help:"Enabled or disabled binding status" choices:"1|0"`
	}
	R(&AlarmTemplateAddToNodeOptions{}, "add-alarmtemplate-to-treenode", "Bind alarm-template to service-tree node", func(s *mcclient.ClientSession, args *AlarmTemplateAddToNodeOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")
		if args.Status > 0 {
			params.Add(jsonutils.NewInt(args.Status), "enabled")
		}

		_, err := modules.AlarmTemplates.PerformAction(s, args.ID, "add-to-treenode", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 从树务树节点删除报警模板
	 */
	type AlarmTemplateDeleteFromNodeOptions struct {
		ID     string `help:"ID of alarm-template"`
		LABELS string `help:"Labels for node"`
	}
	R(&AlarmTemplateDeleteFromNodeOptions{}, "delete-alarmtemplate-from-treenode", "UnBind alarm-template from service-tree node", func(s *mcclient.ClientSession, args *AlarmTemplateDeleteFromNodeOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")

		_, err := modules.AlarmTemplates.PerformAction(s, args.ID, "delete-from-treenode", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 克隆报警模板到服务树节点
	 */
	type AlarmTemplateeCloneToNodeOptions struct {
		ID     string `help:"ID of alarm-template"`
		LABELS string `help:"Labels for node"`
	}
	R(&AlarmTemplateeCloneToNodeOptions{}, "clone-alarmtemplate-to-treenode", "Bind alarm-template to service-tree node", func(s *mcclient.ClientSession, args *AlarmTemplateeCloneToNodeOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")

		_, err := modules.AlarmTemplates.PerformAction(s, args.ID, "clone-to-treenode", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 修改树务树节点绑定的报警模板状态
	 */
	type AlarmTemplateDChangeStatusNodeOptions struct {
		ID     string `help:"ID of alarm-template"`
		LABELS string `help:"Labels for node"`
		STATUS int64  `help:"Enabled or disabled binding status, choices is 1|0"`
	}
	R(&AlarmTemplateDChangeStatusNodeOptions{}, "change-status-alarmtemplate-for-treenode", "UnBind alarm-template from service-tree node", func(s *mcclient.ClientSession, args *AlarmTemplateDChangeStatusNodeOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.LABELS), "node_labels")
		params.Add(jsonutils.NewInt(args.STATUS), "enabled")

		_, err := modules.AlarmTemplates.PerformAction(s, args.ID, "change-bind-status", params)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 修改报警模板下的报警规则的状态
	 */
	type AlarmTemplateAlarmEnabledOptions struct {
		ALARMTEMPLATE_ID string `help:"ID of alarm-template"`
		ALARM_ID         string `help:"ID of alarm"`
		ENABLED          int64  `help:"Enabled or disabled binding status, choices is 1|0"`
	}
	R(&AlarmTemplateAlarmEnabledOptions{}, "change-alarm-enabled", "UnBind alarm-template from service-tree node", func(s *mcclient.ClientSession, args *AlarmTemplateAlarmEnabledOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewInt(args.ENABLED), "enabled")

		_, err := modules.AlarmTemplateAlarms.Update(s, args.ALARMTEMPLATE_ID, args.ALARM_ID, nil, params)
		if err != nil {
			return err
		}
		return nil
	})
}
