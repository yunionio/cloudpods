package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 获得一个全部通信地址组
	 */
	type ContactGroupsListOptions struct {
		options.BaseListOptions
	}
	R(&ContactGroupsListOptions{}, "contact-group-list", "List all contact groups for all the domainsconta", func(s *mcclient.ClientSession, args *ContactGroupsListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}

		result, err := modules.ContactGroups.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ContactGroups.GetColumns(s))
		return nil
	})

}
