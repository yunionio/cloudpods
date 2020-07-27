package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CommonAlertListOptions struct {
		options.BaseListOptions
		// 报警类型
		AlertType string `help:"common alert type" choices:"normal|system"`
		Level     string `help:"common alert notify level" choices:"normal|important|fatal"`
	}
	R(&CommonAlertListOptions{}, "commonalert-list", "List commonalert", func(s *mcclient.ClientSession,
		args *CommonAlertListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.CommonAlertManager.List(s, params)
		if err != nil {
			return nil
		}
		printList(result, modules.CommonAlertManager.GetColumns(s))
		return nil
	})

	type CommonAlertDeleteOptions struct {
		ID    string `help:"ID of alart"`
		Force bool   `help:"force to delete alert"`
	}
	R(&CommonAlertDeleteOptions{}, "commonalert-delete", "List commonalert", func(s *mcclient.ClientSession,
		args *CommonAlertDeleteOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewBool(args.Force), "force")
		object, err := modules.CommonAlertManager.Delete(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(object)
		return nil
	})
}
