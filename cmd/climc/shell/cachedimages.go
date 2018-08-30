package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type CachedImageListOptions struct {
		options.BaseListOptions
	}
	R(&CachedImageListOptions{}, "cached-image-list", "List cached images", func(s *mcclient.ClientSession, args *CachedImageListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
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

	R(&CachedImageShowOptions{}, "cached-image-delete", "Remove cached image information", func(s *mcclient.ClientSession, args *CachedImageShowOptions) error {
		result, err := modules.Cachedimages.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
