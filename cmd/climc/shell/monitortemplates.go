package shell

import (
	"strings"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 创建监控模板
	 */
	type MonitorTemplateCreateOptions struct {
		NAME          string   `help:"The name of the monitor-template"`
		Desc          string   `help:"Description of the monitor-template"`
		MonitorInputs []string `help:"Monitor-inputs add to the monitor-template"`
	}
	R(&MonitorTemplateCreateOptions{}, "monitortemplate-create", "Create or update contact for user", func(s *mcclient.ClientSession, args *MonitorTemplateCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "monitor_template_name")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "monitor_template_desc")
		}
		if len(args.MonitorInputs) > 0 {
			monitors := jsonutils.NewDict()
			for _, f := range args.MonitorInputs {
				parts := strings.Split(f, "=")
				monitors.Add(jsonutils.NewString(parts[1]), parts[0])
			}
			params.Add(monitors, "monitors")
		}

		monitortemplate, err := modules.MonitorTemplates.Create(s, params)

		if err != nil {
			return err
		}

		printObject(monitortemplate)
		return nil
	})

	/**
	 * 列出监控模板
	 */
	type MonitorTemplateListOptions struct {
		BaseListOptions
	}
	R(&MonitorTemplateListOptions{}, "monitortemplate-list", "List all monitor-template", func(s *mcclient.ClientSession, args *MonitorTemplateListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.MonitorTemplates.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.MonitorTemplates.GetColumns(s))
		return nil
	})

	/**
	 * 查看监控模板详情
	 */
	type MonitorTemplateShowOptions struct {
		BaseListOptions
		ID string `help:"The ID of the monitor-template"`
	}
	R(&MonitorTemplateShowOptions{}, "monitortemplate-show", "Show monitor-template", func(s *mcclient.ClientSession, args *MonitorTemplateShowOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.MonitorTemplates.Get(s, args.ID, params)
		if err != nil {
			return err
		}

		printObject(result)
		return nil
	})

	/**
	 * 查看模板下的监控项
	 */
	type MonitorTemplateMonitorInputsListOptions struct {
		MONITOR_TEMPLATE_ID string `help:"The ID of the monitor-template"`
	}
	R(&MonitorTemplateMonitorInputsListOptions{}, "monitortemplate-monitorinput-list", "Get monitor-inputs of a monitor-template", func(s *mcclient.ClientSession, args *MonitorTemplateMonitorInputsListOptions) error {
		result, err := modules.MonitorTemplateInputs.ListDescendent(s, args.MONITOR_TEMPLATE_ID, nil)

		if err != nil {
			return err
		}

		printList(result, modules.MonitorInputs.GetColumns(s))
		return nil
	})

	/**
	 * 更新监控模板信息
	 */
	type MonitorTemplateUpdateOptions struct {
		ID            string   `help:"The ID of the monitor-template"`
		NAME          string   `help:"The name of the monitor-template"`
		Desc          string   `help:"Description of the monitor-template"`
		MonitorInputs []string `help:"Monitor-inputs attached to the monitor-template"`
	}
	R(&MonitorTemplateUpdateOptions{}, "monitortemplate-update", "Update a monitor-template", func(s *mcclient.ClientSession, args *MonitorTemplateUpdateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "monitor_template_name")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "monitor_template_desc")
		}
		if len(args.MonitorInputs) > 0 {
			monitors := jsonutils.NewDict()
			for _, f := range args.MonitorInputs {
				parts := strings.Split(f, "=")
				monitors.Add(jsonutils.NewString(parts[1]), parts[0])
			}
			params.Add(monitors, "monitors")
		}

		monitortemplate, err := modules.MonitorTemplates.Put(s, args.ID, params)

		if err != nil {
			return err
		}

		printObject(monitortemplate)
		return nil
	})

	/**
	 * 向监控模板中添加监控项
	 */
	type MonitorTemplateMonitorInputsOptions struct {
		MONITOR_TEMPLATE_ID string `help:"The ID of the monitor-template"`
		MONITOR_INPUT_NAME  string `help:"The name of the monitor-input"`
		MonitorConfig       string `help:"The config of the monitor-input"`
	}
	R(&MonitorTemplateMonitorInputsOptions{}, "monitortemplate-attach-monitorinput", "Add a monitor-input to a monitor-template", func(s *mcclient.ClientSession, args *MonitorTemplateMonitorInputsOptions) error {
		params := jsonutils.NewDict()
		if len(args.MonitorConfig) > 0 {

			params.Add(jsonutils.NewString(args.MonitorConfig), "monitor_conf_value")
		}

		monitor_template_inputs, err := modules.MonitorTemplateInputs.Update(s, args.MONITOR_TEMPLATE_ID, args.MONITOR_INPUT_NAME, params)

		if err != nil {
			return err
		}

		printObject(monitor_template_inputs)
		return nil
	})

	/**
	 * 从监控模板中删除监控项
	 */
	R(&MonitorTemplateMonitorInputsOptions{}, "monitortemplate-detach-monitorinput", "Add a monitor-input to a monitor-template", func(s *mcclient.ClientSession, args *MonitorTemplateMonitorInputsOptions) error {
		_, err := modules.MonitorInputs.DeleteInContext(s, args.MONITOR_INPUT_NAME, nil, &modules.MonitorTemplates, args.MONITOR_TEMPLATE_ID)
		if err != nil {
			return err
		}
		return nil
	})

	/**
	 * 删除监控模板
	 */
	type MonitorTemplateDeleteOptions struct {
		ID string `help:"The ID of the monitor-template"`
	}
	R(&MonitorTemplateDeleteOptions{}, "monitortemplate-delete", "Delete a monitor-template", func(s *mcclient.ClientSession, args *MonitorTemplateDeleteOptions) error {
		monitortemplate, e := modules.MonitorInputs.Delete(s, args.ID, nil)
		if e != nil {
			return e
		}

		printObject(monitortemplate)
		return nil
	})

}
