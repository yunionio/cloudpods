package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type SchedtagHostListOptions struct {
		BaseListOptions
		Schedtag string `help:"ID or Name of schedtag"`
	}
	R(&SchedtagHostListOptions{}, "schedtag-host-list", "List all scheduler tag and host pairs", func(s *mcclient.ClientSession, args *SchedtagHostListOptions) error {
		mod, err := modules.GetJointModule2(s, &modules.Schedtags, &modules.Hosts)
		if err != nil {
			return err
		}
		params := FetchPagingParams(args.BaseListOptions)
		var result *modules.ListResult
		if len(args.Schedtag) > 0 {
			result, err = mod.ListDescendent(s, args.Schedtag, params)
		} else {
			result, err = mod.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, mod.GetColumns(s))
		return nil
	})

	type SchedtagHostPair struct {
		SCHEDTAG string `help:"Scheduler tag"`
		HOST     string `help:"Host"`
	}
	R(&SchedtagHostPair{}, "schedtag-host-add", "Add a schedtag to a host", func(s *mcclient.ClientSession, args *SchedtagHostPair) error {
		schedtag, err := modules.Schedtaghosts.Attach(s, args.SCHEDTAG, args.HOST, nil)
		if err != nil {
			return err
		}
		printObject(schedtag)
		return nil
	})

	R(&SchedtagHostPair{}, "schedtag-host-remove", "Remove a schedtag from a host", func(s *mcclient.ClientSession, args *SchedtagHostPair) error {
		schedtag, err := modules.Schedtaghosts.Detach(s, args.SCHEDTAG, args.HOST)
		if err != nil {
			return err
		}
		printObject(schedtag)
		return nil
	})
}
