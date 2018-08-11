package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type StoragecacheListOptions struct {
		BaseListOptions

		Manager string `help:"Show regions belongs to the cloud provider"`
	}
	R(&StoragecacheListOptions{}, "storage-cache-list", "List storage caches", func(s *mcclient.ClientSession, args *StoragecacheListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		if len(args.Manager) > 0 {
			params.Add(jsonutils.NewString(args.Manager), "manager")
		}

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

	R(&StoragecacheShowptions{}, "storage-cache-delete", "Delete storage cache", func(s *mcclient.ClientSession, args *StoragecacheShowptions) error {
		result, err := modules.Storagecaches.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type StorageCacheImageActionOptions struct {
		ID    string `help:"ID or name of storage"`
		IMAGE string `help:"ID or name of image"`
		Force bool   `help:"Force refresh cache, even if the image exists in cache"`
	}
	R(&StorageCacheImageActionOptions{}, "storagecache-cache-image", "Ask a storage cache to cache a image", func(s *mcclient.ClientSession, args *StorageCacheImageActionOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IMAGE), "image")
		if args.Force {
			params.Add(jsonutils.JSONTrue, "is_force")
		}
		storage, err := modules.Storagecaches.PerformAction(s, args.ID, "cache-image", params)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})

	type StorageUncacheImageActionOptions struct {
		ID    string `help:"ID or name of storage"`
		IMAGE string `help:"ID or name of image"`
		Force bool   `help:"Force uncache, even if the image exists in cache"`
	}
	R(&StorageUncacheImageActionOptions{}, "storagecache-uncache-image", "Ask a storage cache to remove image from its cache", func(s *mcclient.ClientSession, args *StorageUncacheImageActionOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IMAGE), "image")
		if args.Force {
			params.Add(jsonutils.JSONTrue, "is_force")
		}
		storage, err := modules.Storagecaches.PerformAction(s, args.ID, "uncache-image", params)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})
}
