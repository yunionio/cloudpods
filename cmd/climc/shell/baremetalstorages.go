package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type BaremetalStorageListOptions struct {
		BaseListOptions
		Host string `help:"ID or Name of Host"`
	}
	R(&BaremetalStorageListOptions{}, "baremetal-storage-list", "List baremetal storage pairs", func(s *mcclient.ClientSession, args *BaremetalStorageListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		var result *modules.ListResult
		var err error
		if len(args.Host) > 0 {
			result, err = modules.Baremetalstorages.ListDescendent(s, args.Host, params)
		} else {
			result, err = modules.Baremetalstorages.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Baremetalstorages.GetColumns(s))
		return nil
	})

	type BaremetalStorageDetailOptions struct {
		HOST    string `help:"ID or Name of Host"`
		STORAGE string `help:"ID or Name of Storage"`
	}
	R(&BaremetalStorageDetailOptions{}, "baremetal-storage-show", "Show baremetal storage details", func(s *mcclient.ClientSession, args *BaremetalStorageDetailOptions) error {
		result, err := modules.Baremetalstorages.Get(s, args.HOST, args.STORAGE, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
