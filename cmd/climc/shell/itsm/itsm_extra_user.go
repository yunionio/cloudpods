package itsm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type ExtraUserCreateOptions struct {
		UserName string `help:"user name" required:"true"`
		Pwd      string `help:"password" required:"true"`
		Url      string `help:"url" required:"true"`
		Type     string `help:"extra order type" required:"true"`
	}
	R(&ExtraUserCreateOptions{}, "itsm_extra_user_create", "extra user create",
		func(s *mcclient.ClientSession, args *ExtraUserCreateOptions) error {
			params := jsonutils.Marshal(args)
			ret, err := modules.ExtraUsers.Create(s, params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
	type ExtraUserListOptions struct {
		ID string `help:"ID of user"`
	}
	R(&ExtraUserListOptions{}, "itsm_extra_user_list", "extra user list",
		func(s *mcclient.ClientSession, args *ExtraUserListOptions) error {
			params := jsonutils.NewDict()

			if len(args.ID) > 0 {
				params.Add(jsonutils.NewString(args.ID), "id")
			}
			ret, err := modules.ExtraUsers.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, modules.ExtraUsers.GetColumns(s))
			return nil
		})
}
