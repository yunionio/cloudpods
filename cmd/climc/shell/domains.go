package shell

import (
	"fmt"

	"github.com/yunionio/jsonutils"

	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type DomainListOptions struct {
		Search string `help:"search domain name"`
		Limit  int64  `help:"Items per page" default:"20"`
		Offset int64  `help:"Offset"`
	}
	R(&DomainListOptions{}, "domain-list", "List domains", func(s *mcclient.ClientSession, args *DomainListOptions) error {
		params := jsonutils.NewDict()
		if len(args.Search) > 0 {
			params.Add(jsonutils.NewString(args.Search), "name__icontains")
		}
		if args.Limit > 0 {
			params.Add(jsonutils.NewInt(args.Limit), "limit")
		}
		if args.Offset > 0 {
			params.Add(jsonutils.NewInt(args.Offset), "offset")
		}
		result, err := modules.Domains.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Domains.GetColumns(s))
		return nil
	})

	type DomainDetailOptions struct {
		ID string `help:"ID or domain"`
	}
	R(&DomainDetailOptions{}, "domain-show", "Show detail of domain", func(s *mcclient.ClientSession, args *DomainDetailOptions) error {
		result, err := modules.Domains.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&DomainDetailOptions{}, "domain-delete", "Delete a domain", func(s *mcclient.ClientSession, args *DomainDetailOptions) error {
		objId, err := modules.Domains.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		result, err := modules.Domains.Delete(s, objId, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	R(&DomainDetailOptions{}, "domain-config-show", "Show detail of a domain config", func(s *mcclient.ClientSession, args *DomainDetailOptions) error {
		objId, err := modules.Domains.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		config, err := modules.Domains.GetConfig(s, objId)
		if err != nil {
			return err
		}
		fmt.Println(config.PrettyString())
		return nil
	})
	R(&DomainDetailOptions{}, "domain-config-delete", "Delete a domain config", func(s *mcclient.ClientSession, args *DomainDetailOptions) error {
		objId, err := modules.Domains.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		config, err := modules.Domains.DeleteConfig(s, objId)
		if err != nil {
			return err
		}
		printObject(config)
		return nil
	})
	/* R(&DomainDetailOptions{}, "domain-config-sql", "Config a domain with SQL driver", func(s *mcclient.ClientSession, args *DomainDetailOptions) error {
	    config := jsonutils.NewDict()
	    config.Add(jsonutils.NewString("sql"), "config", "identity", "driver")
	    objId, err := modules.Domains.GetId(s, args.ID, nil)
	    if err != nil {
	        return err
	    }
	    nconf, err := modules.Domains.UpdateConfig(s, objId, config)
	    if err != nil {
	        return err
	    }
	    fmt.Println(nconf.PrettyString())
	    return nil
	}) */

	type DomainConfigLDAPOptions struct {
		ID                      string   `help:"ID of domain to config"`
		URL                     string   `help:"LDAP server URL"`
		SUFFIX                  string   `help:"Suffix"`
		QueryScope              string   `help:"Query scope, either one or sub" choices:"one|sub" default:"sub"`
		PageSize                int      `help:"Page size, default 20" default:"20"`
		User                    string   `help:"User"`
		Password                string   `help:"Password"`
		UserTreeDN              string   `help:"User tree distinguished name"`
		UserFilter              string   `help:"user_filter"`
		UserObjectclass         string   `help:"user_objectclass"`
		UserIdAttribute         string   `help:"user_id_attribute"`
		UserNameAttribute       string   `help:"user_name_attribute"`
		UserEnabledAttribute    string   `help:"user_enabled_attribute"`
		UserEnabledMask         int64    `help:"user_enabled_mask" default:"-1"`
		UserEnabledDefault      string   `help:"user_enabled_default"`
		UserEnabledInvert       string   `help:"user_enabled_invert" choices:"true|false"`
		UserAdditionalAttribute []string `help:"user_additional_attribute_mapping"`
		GroupTreeDN             string   `help:"User tree distinguished name"`
		GroupFilter             string   `help:"group_filter"`
		GroupObjectclass        string   `help:"group_objectclass"`
		GroupIdAttribute        string   `help:"group_id_attribute"`
		GroupNameAttribute      string   `help:"group_name_attribute"`
		GroupMemberAttribute    string   `help:"group_member_attribute"`
		GroupMembersAreIds      bool     `help:"group_members_are_ids"`
	}
	R(&DomainConfigLDAPOptions{}, "domain-config-ldap", "Config a domain with LDAP driver", func(s *mcclient.ClientSession, args *DomainConfigLDAPOptions) error {
		config := jsonutils.NewDict()
		config.Add(jsonutils.NewString("ldap"), "config", "identity", "driver")
		config.Add(jsonutils.NewString(args.URL), "config", "ldap", "url")
		config.Add(jsonutils.NewString(args.SUFFIX), "config", "ldap", "suffix")
		if len(args.QueryScope) > 0 {
			config.Add(jsonutils.NewString(args.QueryScope), "config", "ldap", "query_scope")
		}
		if args.PageSize > 0 {
			config.Add(jsonutils.NewInt(int64(args.PageSize)), "config", "ldap", "page_size")
		}
		if len(args.User) > 0 {
			config.Add(jsonutils.NewString(args.User), "config", "ldap", "user")
		}
		if len(args.Password) > 0 {
			config.Add(jsonutils.NewString(args.Password), "config", "ldap", "password")
		}
		if len(args.UserTreeDN) > 0 {
			config.Add(jsonutils.NewString(args.UserTreeDN), "config", "ldap", "user_tree_dn")
		}
		if len(args.UserFilter) > 0 {
			config.Add(jsonutils.NewString(args.UserFilter), "config", "ldap", "user_filter")
		}
		if len(args.UserObjectclass) > 0 {
			config.Add(jsonutils.NewString(args.UserObjectclass), "config", "ldap", "user_objectclass")
		}
		if len(args.UserIdAttribute) > 0 {
			config.Add(jsonutils.NewString(args.UserIdAttribute), "config", "ldap", "user_id_attribute")
		}
		if len(args.UserNameAttribute) > 0 {
			config.Add(jsonutils.NewString(args.UserNameAttribute), "config", "ldap", "user_name_attribute")
		}
		if len(args.UserEnabledAttribute) > 0 {
			config.Add(jsonutils.NewString(args.UserEnabledAttribute), "config", "ldap", "user_enabled_attribute")
		}
		if args.UserEnabledMask >= 0 {
			config.Add(jsonutils.NewInt(int64(args.UserEnabledMask)), "config", "ldap", "user_enabled_mask")
		}
		if len(args.UserEnabledDefault) > 0 {
			config.Add(jsonutils.NewString(args.UserEnabledDefault), "config", "ldap", "user_enabled_default")
		}
		if len(args.UserEnabledInvert) > 0 {
			if args.UserEnabledInvert == "true" {
				config.Add(jsonutils.JSONTrue, "config", "ldap", "user_enabled_invert")
			} else {
				config.Add(jsonutils.JSONFalse, "config", "ldap", "user_enabled_invert")
			}
		}
		if len(args.UserAdditionalAttribute) > 0 {
			config.Add(jsonutils.NewStringArray(args.UserAdditionalAttribute), "config", "ldap", "user_additional_attribute_mapping")
		}
		if len(args.GroupTreeDN) > 0 {
			config.Add(jsonutils.NewString(args.GroupTreeDN), "config", "ldap", "group_tree_dn")
		}
		if len(args.GroupFilter) > 0 {
			config.Add(jsonutils.NewString(args.GroupFilter), "config", "ldap", "group_filter")
		}
		if len(args.GroupObjectclass) > 0 {
			config.Add(jsonutils.NewString(args.GroupObjectclass), "config", "ldap", "group_objectclass")
		}
		if len(args.GroupIdAttribute) > 0 {
			config.Add(jsonutils.NewString(args.GroupIdAttribute), "config", "ldap", "group_id_attribute")
		}
		if len(args.GroupNameAttribute) > 0 {
			config.Add(jsonutils.NewString(args.GroupNameAttribute), "config", "ldap", "group_name_attribute")
		}
		if len(args.GroupMemberAttribute) > 0 {
			config.Add(jsonutils.NewString(args.GroupMemberAttribute), "config", "ldap", "group_member_attribute")
		}
		if args.GroupMembersAreIds {
			config.Add(jsonutils.JSONTrue, "config", "ldap", "group_members_are_ids")
		}
		objId, err := modules.Domains.GetId(s, args.ID, nil)
		if err != nil {
			return err
		}
		nconf, err := modules.Domains.UpdateConfig(s, objId, config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

	type DomainCreateOptions struct {
		NAME     string `help:"Name of domain"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Set the domain enabled"`
		Disabled bool   `help:"Set the domain disabled"`
	}
	R(&DomainCreateOptions{}, "domain-create", "Create a new domain", func(s *mcclient.ClientSession, args *DomainCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		result, err := modules.Domains.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type DomainUpdateOptions struct {
		ID       string `help:"ID of domain to update"`
		Name     string `help:"Name of domain"`
		Desc     string `help:"Description"`
		Enabled  bool   `help:"Set the domain enabled"`
		Disabled bool   `help:"Set the domain disabled"`
		Driver   string `help:"Set the domain Driver"`
	}
	R(&DomainUpdateOptions{}, "domain-update", "Update a domain", func(s *mcclient.ClientSession, args *DomainUpdateOptions) error {
		obj, err := modules.Domains.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		objId, err := obj.GetString("id")
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Driver) > 0 {
			params.Add(jsonutils.NewString(args.Driver), "driver")
		}

		if args.Enabled && !args.Disabled {
			params.Add(jsonutils.JSONTrue, "enabled")
		} else if !args.Enabled && args.Disabled {
			params.Add(jsonutils.JSONFalse, "enabled")
		}
		result, err := modules.Domains.Patch(s, objId, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
