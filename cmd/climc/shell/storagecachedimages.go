package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type StorageCachedImageListOptions struct {
		BaseListOptions
		Storagecache string `help:"ID or Name of Storage"`
		Image        string `help:"ID or Name of image"`
	}
	R(&StorageCachedImageListOptions{}, "storage-cached-image-list", "List storage cached image pairs", func(s *mcclient.ClientSession, args *StorageCachedImageListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		var result *modules.ListResult
		var err error
		if len(args.Storagecache) > 0 {
			result, err = modules.Storagecachedimages.ListDescendent(s, args.Storagecache, params)
		} else if len(args.Image) > 0 {
			result, err = modules.Storagecachedimages.ListDescendent2(s, args.Image, params)
		} else {
			result, err = modules.Storagecachedimages.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Storagecachedimages.GetColumns(s))
		return nil
	})

	type StorageCachedImageUpdateOptions struct {
		STORAGECACHE string `help:"ID or Name of Storage cache"`
		IMAGE        string `help:"ID or name of image"`
		Status       string `help:"Status"`
	}
	R(&StorageCachedImageUpdateOptions{}, "storage-cached-image-update", "Update storage cached image", func(s *mcclient.ClientSession, args *StorageCachedImageUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Status) > 0 {
			params.Add(jsonutils.NewString(args.Status), "status")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Storagecachedimages.Update(s, args.STORAGECACHE, args.IMAGE, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type StorageCachedImageShowOptions struct {
		STORAGECACHE string `help:"ID or Name of Storage cache"`
		IMAGE        string `help:"ID or name of image"`
	}
	R(&StorageCachedImageShowOptions{}, "storage-cached-image-show", "Show storage cached image", func(s *mcclient.ClientSession, args *StorageCachedImageShowOptions) error {
		result, err := modules.Storagecachedimages.Get(s, args.STORAGECACHE, args.IMAGE, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
