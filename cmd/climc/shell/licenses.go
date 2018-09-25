package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type LicenseListOptions struct {
		options.BaseListOptions
	}

	R(&LicenseListOptions{}, "licenses-list", "show licenses", func(s *mcclient.ClientSession, args *LicenseListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}

		lics, err := modules.License.List(s, params)
		if err != nil {
			return err
		}

		printList(lics, modules.License.GetColumns(s))
		return nil
	})

	type LicenseShowOptions struct {
		SERVICE string `help:"service name"  choices:"compute|service_tree"`
	}

	R(&LicenseShowOptions{}, "licenses-show", "show actived license", func(s *mcclient.ClientSession, args *LicenseShowOptions) error {
		lic, e := modules.License.Get(s, args.SERVICE, nil)
		if e != nil {
			return e
		}

		printObject(lic)
		return nil
	})

	type LicenseStatusOptions struct {
		SERVICE string `help:"service name"  choices:"compute|service_tree"`
	}

	R(&LicenseStatusOptions{}, "licenses-usage", "show license usages status", func(s *mcclient.ClientSession, args *LicenseStatusOptions) error {
		status, err := modules.License.GetSpecific(s, args.SERVICE, "status", nil)
		if err != nil {
			return err
		}

		printObject(status)
		return nil
	})

}
