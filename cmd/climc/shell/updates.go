package shell

import (
	"fmt"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SUpdateListOptions struct {
		options.BaseListOptions
		Region string `help:"cloud region ID or Name"`
	}

	R(&SUpdateListOptions{}, "update-list", "List updates", func(s *mcclient.ClientSession, args *SUpdateListOptions) error {
		// TODO filer by region
		result, err := modules.Updates.List(s, nil)

		if err != nil {
			return err
		}

		printList(result, modules.Updates.GetColumns(s))
		return nil
	})

	type SUpdatePerformOptions struct {
		Cmp     bool `help:"update Controller And all the Compute nodes automatically"`
		CmpOnly bool `help:"Updates only computes nodes, excluding controller nodes"`
	}

	R(&SUpdatePerformOptions{}, "update-perform", "Update the Controler", func(s *mcclient.ClientSession, args *SUpdatePerformOptions) error {
		params := jsonutils.NewDict()

		if args.Cmp && args.CmpOnly {
			return fmt.Errorf("--cmp and --cmp-only can't go together")
		}

		if args.Cmp {
			params.Add(jsonutils.JSONTrue, "cmp")
		} else if args.CmpOnly {
			params.Add(jsonutils.JSONTrue, "cmp_only")
		}

		result, err := modules.Updates.List(s, nil)

		if err != nil {
			return err
		}
		modules.Updates.DoUpdate(s, params)
		printList(result, modules.Updates.GetColumns(s))
		return nil
	})
}
