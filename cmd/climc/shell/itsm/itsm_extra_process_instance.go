package itsm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type ExtraProcessInstanceListOptions struct {
		ID     string `help:"ID of ExtraProcessInstance"`
		UserId string `help:"ID of user"`
	}
	R(&ExtraProcessInstanceListOptions{}, "itsm_extra_process_instance_list", "extra process instance list",
		func(s *mcclient.ClientSession, args *ExtraProcessInstanceListOptions) error {
			params := jsonutils.NewDict()
			if len(args.ID) > 0 {
				params.Add(jsonutils.NewString(args.ID), "id")
			}
			if len(args.UserId) > 0 {
				params.Add(jsonutils.NewString(args.UserId), "user_id")
			}

			ret, err := modules.ExtraProcessInstance.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, modules.ExtraProcessInstance.GetColumns(s))
			return nil
		})

	type ExtraProcessInstanceShowOptions struct {
		ID string `help:"ID of ExtraProcessInstance" required:"true"`
	}
	R(&ExtraProcessInstanceShowOptions{}, "itsm_extra_process_instance_list", "extra process instance list",
		func(s *mcclient.ClientSession, args *ExtraProcessInstanceShowOptions) error {
			ret, err := modules.ExtraProcessInstance.Get(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
}
