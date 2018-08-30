package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type StorageListOptions struct {
		options.BaseListOptions
		Share  bool   `help:"Share storage list"`
		Local  bool   `help:"Local storage list"`
		Usable bool   `help:"Usable storage list"`
		Zone   string `help:"List storages in zone"`
		Region string `help:"List storages in region"`

		Manager string `help:"Show regions belongs to the cloud provider"`
	}
	R(&StorageListOptions{}, "storage-list", "List storages", func(s *mcclient.ClientSession, args *StorageListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if args.Share {
			params.Add(jsonutils.JSONTrue, "share")
		}
		if args.Local {
			params.Add(jsonutils.JSONTrue, "local")
		}
		if args.Usable {
			params.Add(jsonutils.JSONTrue, "usable")
		}
		if len(args.Region) > 0 {
			params.Add(jsonutils.NewString(args.Region), "region")
		}

		if len(args.Manager) > 0 {
			params.Add(jsonutils.NewString(args.Manager), "manager")
		}

		var result *modules.ListResult
		var err error
		if len(args.Zone) > 0 {
			result, err = modules.Storages.ListInContext(s, params, &modules.Zones, args.Zone)
		} else {
			result, err = modules.Storages.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Storages.GetColumns(s))
		return nil
	})

	type StorageUpdateOptions struct {
		ID          string  `help:"ID or Name of storage to update"`
		Name        string  `help:"New Name of storage"`
		Desc        string  `help:"Description" metavar:"<DESCRIPTION>"`
		CommitBound float64 `help:"Upper bound of storage overcommit rate"`
		StorageType string  `help:"Storage type" choices:"local|nas|vsan|rbd|baremetal"`
		MediumType  string  `help:"Medium type, either ssd or rotate" choices:"ssd|rotate"`
		Reserved    string  `help:"Reserved storage space"`
	}
	R(&StorageUpdateOptions{}, "storage-update", "Update a storage", func(s *mcclient.ClientSession, args *StorageUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.CommitBound > 0 {
			params.Add(jsonutils.NewFloat(args.CommitBound), "cmtbound")
		}
		if len(args.StorageType) > 0 {
			params.Add(jsonutils.NewString(args.StorageType), "storage_type")
		}
		if len(args.MediumType) > 0 {
			params.Add(jsonutils.NewString(args.MediumType), "medium_type")
		}
		if len(args.Reserved) > 0 {
			params.Add(jsonutils.NewString(args.Reserved), "reserved")
		}
		result, err := modules.Storages.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type StorageCreateOptions struct {
		NAME        string `help:"Name of the Storage"`
		Capacity    int64  `help:"Capacity of the Storage"`
		MediumType  string `help:"Medium type, either ssd or rotate" choices:"ssd|rotate"`
		StorageType string `help:"Storage type" choices:"local|nas|vsan|rbd|baremetal"`
		MonHost     string `helo:"Ceph mon_host config"`
		Key         string `helo:"Ceph key config"`
		Pool        string `helo:"Ceph Poll Name"`
	}
	R(&StorageCreateOptions{}, "storage-create", "Create a Storage", func(s *mcclient.ClientSession, args *StorageCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewInt(args.Capacity), "capacity")
		params.Add(jsonutils.NewString(args.StorageType), "storage_type")
		params.Add(jsonutils.NewString(args.MediumType), "medium_type")
		if args.StorageType == "rbd" {
			if args.MonHost == "" || args.Key == "" || args.Pool == "" {
				return fmt.Errorf("Not enough arguments, missing mon_host、key or pool")
			}
			params.Add(jsonutils.NewString(args.MonHost), "rbd_mon_host")
			params.Add(jsonutils.NewString(args.Key), "rbd_key")
			params.Add(jsonutils.NewString(args.Pool), "rbd_pool")
		}
		storage, err := modules.Storages.Create(s, params)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})

	type StorageShowOptions struct {
		ID string `help:"ID or Name of the storage to show"`
	}
	R(&StorageShowOptions{}, "storage-show", "Show storage details", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-delete", "Delete a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-enable", "Enable a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-disable", "Disable a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-enable", "Enable a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-disable", "Disable a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.PerformAction(s, args.ID, "disable", nil)
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
	R(&StorageCacheImageActionOptions{}, "storage-cache-image", "Ask a storage to cache a image", func(s *mcclient.ClientSession, args *StorageCacheImageActionOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IMAGE), "image")
		if args.Force {
			params.Add(jsonutils.JSONTrue, "is_force")
		}
		storage, err := modules.Storages.PerformAction(s, args.ID, "cache-image", params)
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
	R(&StorageUncacheImageActionOptions{}, "storage-uncache-image", "Ask a storage to remove image from its cache", func(s *mcclient.ClientSession, args *StorageUncacheImageActionOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IMAGE), "image")
		if args.Force {
			params.Add(jsonutils.JSONTrue, "is_force")
		}
		storage, err := modules.Storages.PerformAction(s, args.ID, "uncache-image", params)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})

}
