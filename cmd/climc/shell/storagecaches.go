package shell

import (
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

func init() {
	type StoragecacheListOptions struct {
		BaseListOptions
	}
	R(&StoragecacheListOptions{}, "storage-cache-list", "List storage caches", func(s *mcclient.ClientSession, args *StoragecacheListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		result, err := modules.Storagecaches.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Storagecaches.GetColumns(s))
		return nil
	})

	type StoragecacheShowptions struct {
		ID string `help:"ID or Name of storagecache"`
	}
	R(&StoragecacheShowptions{}, "storage-cache-show", "Show details of storage caches", func(s *mcclient.ClientSession, args *StoragecacheShowptions) error {
		result, err := modules.Storagecaches.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
