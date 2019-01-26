package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 创建一条报警规则
	 */
	type MeterAlertCreateOptions struct {
		TYPE       string  `help:"Alert rule type" choices:"balance|resFee"`
		PROVIDER   string  `help:"Name of the cloud platform"`
		ACCOUNT_ID string  `help:"ID of the cloud platform"`
		THRESHOLD  float64 `help:"Threshold value of the metric"`
		COMPARATOR string  `help:"Comparison operator for join expressions" choices:">|<|>=|<=|=|!="`
		RECIPIENTS string  `help:"Comma separated recipient ID"`
		LEVEL      string  `help:"Alert level" choices:"normal|important|fatal"`
		CHANNEL    string  `help:"Ways to send an alarm" choices:"email|mobile"`
	}
	R(&MeterAlertCreateOptions{}, "meteralert-create", "Create a meter alert rule", func(s *mcclient.ClientSession, args *MeterAlertCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TYPE), "type")
		params.Add(jsonutils.NewString(args.PROVIDER), "provider")
		params.Add(jsonutils.NewString(args.ACCOUNT_ID), "account_id")
		params.Add(jsonutils.NewFloat(args.THRESHOLD), "threshold")
		params.Add(jsonutils.NewString(args.COMPARATOR), "comparator")
		params.Add(jsonutils.NewString(args.RECIPIENTS), "recipients")
		params.Add(jsonutils.NewString(args.LEVEL), "level")
		params.Add(jsonutils.NewString(args.CHANNEL), "channel")

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
		options.BaseListOptions
	}
	R(&MeterAlertListOptions{}, "meteralert-list", "List meter alert", func(s *mcclient.ClientSession, args *MeterAlertListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.MeterAlert.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.MeterAlert.GetColumns(s))
		return nil
	})
}
