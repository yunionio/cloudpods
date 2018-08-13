package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 获得一个全部通信地址组
	 */
	type ContactGroupsListOptions struct {
		BaseListOptions
	}
	R(&ContactGroupsListOptions{}, "contact-group-list", "List all contact groups for all the domainsconta", func(s *mcclient.ClientSession, args *ContactGroupsListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.ContactGroups.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.ContactGroups.GetColumns(s))
		return nil
	})

}
