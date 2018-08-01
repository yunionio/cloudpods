package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type SecGroupRulesListOptions struct {
		BaseListOptions
		Secgroup  string `help:"Secgroup ID or Name"`
		Direction string `help:"filter Direction of rule" choices:"in|out"`
		Protocol  string `help:"filter Protocol of rule" choices:"any|tcp|udp|icmp"`
		Action    string `help:"filter Actin of rule" choices:"allow|deny"`
	}

	R(&SecGroupRulesListOptions{}, "secgroup-rule-list", "List all security group", func(s *mcclient.ClientSession, args *SecGroupRulesListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		if len(args.Secgroup) > 0 {
			params.Add(jsonutils.NewString(args.Secgroup), "secgroup")
		}
		if len(args.Direction) > 0 {
			params.Add(jsonutils.NewString(args.Direction), "direction")
		}
		if len(args.Protocol) > 0 {
			params.Add(jsonutils.NewString(args.Protocol), "protocol")
		}
		if len(args.Action) > 0 {
			params.Add(jsonutils.NewString(args.Action), "action")
		}
		result, err := modules.SecGroupRules.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.SecGroupRules.GetColumns(s))
		return nil
	})

	type SecGroupRuleDetailOptions struct {
		ID string `help:"ID or Name of security group rule"`
	}
	R(&SecGroupRuleDetailOptions{}, "secgroup-rule-show", "Show details of rule", func(s *mcclient.ClientSession, args *SecGroupRuleDetailOptions) error {
		if rule, e := modules.SecGroupRules.Get(s, args.ID, nil); e != nil {
			return e
		} else {
			printObject(rule)
		}
		return nil
	})

	R(&SecGroupRuleDetailOptions{}, "secgroup-rule-delete", "Delete a disk", func(s *mcclient.ClientSession, args *SecGroupRuleDetailOptions) error {
		if rule, e := modules.SecGroupRules.Delete(s, args.ID, nil); e != nil {
			return e
		} else {
			printObject(rule)
		}
		return nil
	})

	type SecGroupRulesCreateOptions struct {
		NAME      string `help:"Name of security group rule to create"`
		SECGROUP  string `help:"Secgroup ID or Name" metavar:"Secgroup"`
		Direction string `help:"Direction of rule" choices:"in|out"`
		Action    string `help:"Action of rule" choices:"allow|deny"`
		Protocol  string `help:"Protocol of rule" choices:"tcp|udp|icmp|any"`
		Ports     string `help:"Ports of rule"`
		Cidr      string `help:"Cidr of rule"`
		Priority  int64  `help:"priority of Rule"`
		Desc      string `help:"Description"`
	}

	R(&SecGroupRulesCreateOptions{}, "secgroup-rule-create", "Create all security group rule", func(s *mcclient.ClientSession, args *SecGroupRulesCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Priority > 0 {
			params.Add(jsonutils.NewInt(args.Priority), "priority")
		}
		if len(args.Direction) > 0 {
			params.Add(jsonutils.NewString(args.Direction), "direction")
		}
		if len(args.Action) > 0 {
			params.Add(jsonutils.NewString(args.Action), "action")
		}
		if len(args.Protocol) > 0 {
			params.Add(jsonutils.NewString(args.Protocol), "protocol")
		}
		if len(args.Ports) > 0 {
			params.Add(jsonutils.NewString(args.Ports), "ports")
		}
		if len(args.Cidr) > 0 {
			params.Add(jsonutils.NewString(args.Cidr), "cidr")
		}
		params.Add(jsonutils.NewString(args.SECGROUP), "secgroup")
		secgrouprules, err := modules.SecGroupRules.Create(s, params)
		if err != nil {
			return err
		}
		printObject(secgrouprules)
		return nil
	})

	type SecGroupRulesUpdateOptions struct {
		ID       string `help:"ID or name of rule"`
		Name     string `help:"New name of rule"`
		Priority int64  `help:"priority of Rule"`
		Protocol string `help:"Protocol of rule" choices:"any|tcp|udp|icmp"`
		Ports    string `help:"Ports of rule"`
		Cidr     string `help:"Cidr of rule"`
		Action   string `help:"filter Actin of rule" choices:"allow|deny"`
		Desc     string `help:"Description" metavar:"Description"`
	}

	R(&SecGroupRulesUpdateOptions{}, "secgroup-rule-update", "Update property of a security group rule", func(s *mcclient.ClientSession, args *SecGroupRulesUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.Priority > 0 {
			params.Add(jsonutils.NewInt(args.Priority), "priority")
		}
		if len(args.Protocol) > 0 {
			params.Add(jsonutils.NewString(args.Protocol), "protocol")
		}
		if len(args.Ports) > 0 {
			params.Add(jsonutils.NewString(args.Ports), "ports")
		}
		if len(args.Cidr) > 0 {
			params.Add(jsonutils.NewString(args.Cidr), "cidr")
		}
		if len(args.Action) > 0 {
			params.Add(jsonutils.NewString(args.Action), "action")
		}
		if rule, e := modules.SecGroupRules.Update(s, args.ID, params); e != nil {
			return e
		} else {
			printObject(rule)
		}
		return nil
	})
}
