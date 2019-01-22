package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CachedImageListOptions struct {
		options.BaseListOptions
		ImageType string `help:"image type" choices:"system|customized|shared|market"`

		Region string `help:"show images cached at cloud region"`
		Zone   string `help:"show images cached at zone"`
	}
	R(&CachedImageListOptions{}, "cached-image-list", "List cached images", func(s *mcclient.ClientSession, args *CachedImageListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Cachedimages.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Cachedimages.GetColumns(s))
		return nil
	})

	type CachedImageShowOptions struct {
		ID string `help:"ID or Name of the cached image to show"`
	}
	R(&CachedImageShowOptions{}, "cached-image-show", "Show cached image details", func(s *mcclient.ClientSession, args *CachedImageShowOptions) error {
		result, err := modules.Cachedimages.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&CachedImageShowOptions{}, "cached-image-refresh", "Refresh cached image details", func(s *mcclient.ClientSession, args *CachedImageShowOptions) error {
		result, err := modules.Cachedimages.PerformAction(s, args.ID, "refresh", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type CachedImageDeleteOptions struct {
		ID []string `help:"ID or Name of the cached image to show"`
	}
	R(&CachedImageDeleteOptions{}, "cached-image-delete", "Remove cached image information", func(s *mcclient.ClientSession, args *CachedImageDeleteOptions) error {
		results := modules.Cachedimages.BatchDelete(s, args.ID, nil)
		printBatchResults(results, modules.Cachedimages.GetColumns(s))
		return nil
	})
}
