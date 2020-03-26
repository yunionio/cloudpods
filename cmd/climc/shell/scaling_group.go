package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ScalingGroupListOptions struct {
		options.BaseListOptions
		Hypervisor  string `help:"Hypervisor"`
		Cloudregion string `help:"Cloudregion"`
		Network     string `help:"network"`
	}
	R(&ScalingGroupListOptions{}, "scaling-group-list", "List scaling group", func(s *mcclient.ClientSession, args *ScalingGroupListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.ScalingGroup.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ScalingGroup.GetColumns(s))
		return nil
	})
	type ScalingGroupCreateOptions struct {
		NAME string

		Hypervisor           string
		Cloudregion          string
		Network              string
		GuestTemplate        string
		MinInstanceNumber    string
		MaxInstanceNumber    string
		DesireInstanceNumber string
		Loadbalance          string
	}
	R(&ScalingGroupCreateOptions{}, "scaling-group-create", "Create scaling group", func(s *mcclient.ClientSession, args *ScalingGroupCreateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		scalingGroup, err := modules.ScalingGroup.Create(s, params)
		if err != nil {
			return err
		}
		printObject(scalingGroup)
		return nil
	})
	type ScalingGroupDeleteOptions struct {
		ID string `help:"ScalingGroup ID or Name"`
	}
	R(&ScalingGroupDeleteOptions{}, "scaling-group-delete", "Delete Scaling Group", func(s *mcclient.ClientSession,
		args *ScalingGroupDeleteOptions) error {
		scalingGroup, err := modules.ScalingGroup.Delete(s, args.ID, jsonutils.NewDict())
		if err != nil {
			return err
		}
		printObject(scalingGroup)
		return nil
	})
	type ScalingGroupListInstanceOptions struct {
		ID string `help:"ScalingGroup ID or Name"`
	}
	R(&ScalingGroupListInstanceOptions{}, "scaling-group-list-instance", "List all instance of Scaling Group",
		func(s *mcclient.ClientSession, args *ScalingGroupListInstanceOptions) error {
			params := jsonutils.NewDict()
			params.Set("scaling_group", jsonutils.NewString(args.ID))
			servers, err := modules.Servers.List(s, params)
			if err != nil {
				return err
			}
			printList(servers, modules.Servers.GetColumns(s))
			return nil
		},
	)
	type ScalingGroupRemoveInstanceOptions struct {
		ID     string `help:"ScalingGroup ID or Name"`
		SERVER string `help:"Server ID or Name"`
		Delete bool   `help:"If delete the server"`
	}
	R(&ScalingGroupRemoveInstanceOptions{}, "scaling-group-remove-instance", "Remove instance of ScalingGroup",
		func(s *mcclient.ClientSession, args *ScalingGroupRemoveInstanceOptions) error {
			params := jsonutils.NewDict()
			params.Set("scaling_group", jsonutils.NewString(args.ID))
			params.Set("delete_server", jsonutils.NewBool(args.Delete))
			server, err := modules.Servers.PerformAction(s, args.SERVER, "detach-scaling-group", params)
			if err != nil {
				return err
			}
			printObject(server)
			return nil
		},
	)
	type ScalingGroupEnableOptions struct {
		ID string `help:"ScalingGroup ID or Name"`
	}
	R(&ScalingGroupEnableOptions{}, "scaling-group-enable", "Enable ScalingGroup", func(s *mcclient.ClientSession,
		args *ScalingGroupEnableOptions) error {
		ret, err := modules.ScalingGroup.PerformAction(s, args.ID, "enable", jsonutils.NewDict())
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&ScalingGroupEnableOptions{}, "scaling-group-disable", "Disable ScalingGroup", func(s *mcclient.ClientSession,
		args *ScalingGroupEnableOptions) error {
		ret, err := modules.ScalingGroup.PerformAction(s, args.ID, "disable", jsonutils.NewDict())
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
