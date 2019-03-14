package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type BaremetalAgentListOptions struct {
		options.BaseListOptions
	}
	R(&BaremetalAgentListOptions{}, "baremetal-agent-list", "List baremetal agent", func(s *mcclient.ClientSession, args *BaremetalAgentListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Baremetalagents.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Baremetalagents.GetColumns(s))
		return nil
	})

	type BaremetalAgentOpsOperations struct {
		ID string `help:"ID or name of agent"`
	}
	R(&BaremetalAgentOpsOperations{}, "baremetal-agent-enable", "Enable baremetal agent", func(s *mcclient.ClientSession, args *BaremetalAgentOpsOperations) error {
		result, err := modules.Baremetalagents.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&BaremetalAgentOpsOperations{}, "baremetal-agent-disable", "Disable baremetal agent", func(s *mcclient.ClientSession, args *BaremetalAgentOpsOperations) error {
		result, err := modules.Baremetalagents.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
