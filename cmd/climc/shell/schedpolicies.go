package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SchedpoliciesListOptions struct {
		options.BaseListOptions
	}
	R(&SchedpoliciesListOptions{}, "sched-policy-list", "List scheduler policies", func(s *mcclient.ClientSession, args *SchedpoliciesListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err
			}
		}
		results, err := modules.Schedpolicies.List(s, params)
		if err != nil {
			return err
		}
		printList(results, modules.Schedpolicies.GetColumns(s))
		return nil
	})

	type SchedpoliciesCreateOptions struct {
		NAME      string `help:"name of the sched policy"`
		STRATEGY  string `help:"strategy for the schedtag" choices:"require|prefer|avoid|exclude"`
		SCHEDTAG  string `help:"ID or name of schedtag"`
		CONDITION string `help:"condition that assign schedtag to hosts"`
		Enable    bool   `help:"create the policy with enabled status"`
		Disable   bool   `help:"create the policy with disabled status"`
	}
	R(&SchedpoliciesCreateOptions{}, "sched-policy-create", "create a sched policty", func(s *mcclient.ClientSession, args *SchedpoliciesCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.STRATEGY), "strategy")
		params.Add(jsonutils.NewString(args.CONDITION), "condition")
		params.Add(jsonutils.NewString(args.SCHEDTAG), "schedtag")

		if args.Enable {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if args.Disable {
			params.Add(jsonutils.JSONFalse, "disabled")
		}

		result, err := modules.Schedpolicies.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SchedpoliciesUpdateOptions struct {
		ID        string `help:"ID or name of the sched policy"`
		Name      string `help:"new name of sched policy"`
		Strategy  string `help:"schedtag strategy" choices:"require|prefer|avoid|exclude"`
		SchedTag  string `help:"ID or name of schedtag"`
		Condition string `help:"condition that assign schedtag to hosts"`
		Enable    bool   `help:"make the sched policy enabled"`
		Disable   bool   `help:"make the sched policy disabled"`
	}
	R(&SchedpoliciesUpdateOptions{}, "sched-policy-update", "update a sched policy", func(s *mcclient.ClientSession, args *SchedpoliciesUpdateOptions) error {
		params := jsonutils.NewDict()

		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Strategy) > 0 {
			params.Add(jsonutils.NewString(args.Strategy), "strategy")
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
		result, err := modules.Schedpolicies.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SchedpoliciesDeleteOptions struct {
		ID string `help:"ID or name of the sched policy"`
	}
	R(&SchedpoliciesDeleteOptions{}, "sched-policy-delete", "delete a sched policy", func(s *mcclient.ClientSession, args *SchedpoliciesDeleteOptions) error {
		result, err := modules.Schedpolicies.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SchedpoliciesEvaluateOptions struct {
		ID     string `help:"ID or name of the sched policy"`
		SERVER string `help:"ID or name of the server"`
	}
	R(&SchedpoliciesEvaluateOptions{}, "sched-policy-evaluate", "Evaluate sched policy", func(s *mcclient.ClientSession, args *SchedpoliciesEvaluateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.SERVER), "server")
		result, err := modules.Schedpolicies.PerformAction(s, args.ID, "evaluate", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
