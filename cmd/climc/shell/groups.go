package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type GroupListOptions struct {
		Name   string `help:"Name of the groups to filter"`
		Domain string `help:"Domain to filter"`
		Limit  int64  `help:"Items per page" default:"20"`
		Offset int64  `help:"Offset"`
		Search string `help:"search text"`
	}
	R(&GroupListOptions{}, "group-list", "List groups", func(s *mcclient.ClientSession, args *GroupListOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Domain) > 0 {
			domainId, e := modules.Domains.GetId(s, args.Domain, nil)
			if e != nil {
				return e
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		if args.Limit > 0 {
			params.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			params.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		if len(args.Search) > 0 {
			params.Add(jsonutils.NewString(args.Search), "name__icontains")
		}
		result, err := modules.Groups.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Groups.GetColumns(s))
		return nil
	})

	type GroupShowOptions struct {
		ID     string `help:"ID or Name of group"`
		Domain string `help:"Id or Name of domain"`
	}
	R(&GroupShowOptions{}, "group-show", "Show details of a group", func(s *mcclient.ClientSession, args *GroupShowOptions) error {
		params := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		grpId, err := modules.Groups.GetId(s, args.ID, params)
		if err != nil {
			return err
		}
		result, err := modules.Groups.GetById(s, grpId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&GroupShowOptions{}, "group-user-list", "Show members of a group", func(s *mcclient.ClientSession, args *GroupShowOptions) error {
		params := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		grpId, err := modules.Groups.GetId(s, args.ID, params)
		if err != nil {
			return err
		}
		users, err := modules.Groups.GetUsers(s, grpId)
		if err != nil {
			return err
		}
		printList(users, modules.UsersV3.GetColumns(s))
		return nil
	})

	type GroupCreateOptions struct {
		NAME   string `help:"Name of the group"`
		Desc   string `help:"Description"`
		Domain string `help:"Domain ID or Name"`
	}
	R(&GroupCreateOptions{}, "group-create", "Create a group", func(s *mcclient.ClientSession, args *GroupCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Domain) > 0 {
			domainId, e := modules.Domains.GetId(s, args.Domain, nil)
			if e != nil {
				return e
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		result, e := modules.Groups.Create(s, params)
		if e != nil {
			return e
		}
		printObject(result)
		return nil
	})

	R(&GroupShowOptions{}, "group-delete", "Delete a group", func(s *mcclient.ClientSession, args *GroupShowOptions) error {
		params := jsonutils.NewDict()
		if len(args.Domain) > 0 {
			domainId, err := modules.Domains.GetId(s, args.Domain, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(domainId), "domain_id")
		}
		grpId, err := modules.Groups.GetId(s, args.ID, params)
		if err != nil {
			return err
		}
		result, err := modules.Groups.Delete(s, grpId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
