package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type DynamicschedtagListOptions struct {
		options.BaseListOptions
	}
	R(&DynamicschedtagListOptions{}, "dynamic-schedtag-list", "List dynamic schedtag conditions", func(s *mcclient.ClientSession, args *DynamicschedtagListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
		}
		results, err := modules.Dynamicschedtags.List(s, params)
		if err != nil {
			return err
		}
		printList(results, modules.Dynamicschedtags.GetColumns(s))
		return nil
	})

	type DynamicschedtagCreateOptions struct {
		NAME      string `help:"name of the dynamic schedtag"`
		SCHEDTAG  string `help:"ID or name of schedtag"`
		CONDITION string `help:"condition that assign schedtag to hosts"`
		Enable    bool   `help:"create the policy with enabled status"`
		Disable   bool   `help:"create the policy with disabled status"`
	}
	R(&DynamicschedtagCreateOptions{}, "dynamic-schedtag-create", "create dynamic schedtag", func(s *mcclient.ClientSession, args *DynamicschedtagCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.CONDITION), "condition")
		params.Add(jsonutils.NewString(args.SCHEDTAG), "schedtag")

		if args.Enable {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if args.Disable {
			params.Add(jsonutils.JSONFalse, "enabled")
		}

		result, err := modules.Dynamicschedtags.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DynamicschedtagUpdateOptions struct {
		ID        string `help:"ID or name of the dynamic schedtag"`
		Name      string `help:"new name of the dynamic schedtag"`
		SchedTag  string `help:"ID or name of schedtag"`
		Condition string `help:"condition that assign schedtag to hosts"`
		Enable    bool   `help:"update to enabled"`
		Disable   bool   `help:"update to disabled"`
	}
	R(&DynamicschedtagUpdateOptions{}, "dynamic-schedtag-update", "update dynamic schedtag", func(s *mcclient.ClientSession, args *DynamicschedtagUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Condition) > 0 {
			params.Add(jsonutils.NewString(args.Condition), "condition")
		}
		if len(args.SchedTag) > 0 {
			params.Add(jsonutils.NewString(args.SchedTag), "schedtag")
		}
		if args.Enable {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if args.Disable {
			params.Add(jsonutils.JSONFalse, "enabled")
		}

		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Dynamicschedtags.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DynamicschedtagDeleteOptions struct {
		ID string `help:"ID or name of the dynamic schedtag"`
	}
	R(&DynamicschedtagDeleteOptions{}, "dynamic-schedtag-delete", "delete dynamic schedtag", func(s *mcclient.ClientSession, args *DynamicschedtagDeleteOptions) error {
		result, err := modules.Dynamicschedtags.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&DynamicschedtagDeleteOptions{}, "dynamic-schedtag-show", "Show details of a dyanmic schedtag policy", func(s *mcclient.ClientSession, args *DynamicschedtagDeleteOptions) error {
		result, err := modules.Dynamicschedtags.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DynamicSchedtagEvaluateOptions struct {
		ID     string `help:"ID or name of the sched policy"`
		HOST   string `help:"ID or name of the host"`
		SERVER string `help:"ID or name of the server"`
	}
	R(&DynamicSchedtagEvaluateOptions{}, "dynamic-schedtag-evaluate", "Evaluate dynamic schedtag condition", func(s *mcclient.ClientSession, args *DynamicSchedtagEvaluateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.HOST), "host")
		params.Add(jsonutils.NewString(args.SERVER), "server")
		result, err := modules.Dynamicschedtags.PerformAction(s, args.ID, "evaluate", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
