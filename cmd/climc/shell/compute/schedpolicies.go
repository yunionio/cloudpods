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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SchedpoliciesListOptions struct {
		options.BaseListOptions
	}
	R(&SchedpoliciesListOptions{}, "sched-policy-list", "List scheduler policies", func(s *mcclient.ClientSession, args *SchedpoliciesListOptions) error {
		var params *jsonutils.JSONDict
		{
			param, err := args.BaseListOptions.Params()
			if err != nil {
				return err
			}
			params = param.(*jsonutils.JSONDict)
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

	type SchedpoliciesShowOptions struct {
		ID string `help:"ID or name of the sched policy"`
	}
	R(&SchedpoliciesShowOptions{}, "sched-policy-show", "show details of a sched policy", func(s *mcclient.ClientSession, args *SchedpoliciesShowOptions) error {
		result, err := modules.Schedpolicies.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SchedpoliciesEvaluateOptions struct {
		ID           string `help:"ID or name of the sched policy"`
		OBJECT       string `help:"ID or name of the object"`
		ResourceType string `help:"Resource type of the object" default:"server" choices:"server|disk" short-token:"t"`
	}
	R(&SchedpoliciesEvaluateOptions{}, "sched-policy-evaluate", "Evaluate sched policy", func(s *mcclient.ClientSession, args *SchedpoliciesEvaluateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.OBJECT), "object")
		params.Add(jsonutils.NewString(args.ResourceType), "resource_type")
		result, err := modules.Schedpolicies.PerformAction(s, args.ID, "evaluate", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
