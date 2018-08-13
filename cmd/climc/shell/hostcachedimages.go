package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type HostCachedImageListOptions struct {
		BaseListOptions
		Host  string `help:"ID or Name of Host"`
		Image string `help:"ID or Name of image"`
	}
	R(&HostCachedImageListOptions{}, "host-cachedimage-list", "List host cached image pairs", func(s *mcclient.ClientSession, args *HostCachedImageListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		var result *modules.ListResult
		var err error
		if len(args.Host) > 0 {
			result, err = modules.Hostcachedimages.ListDescendent(s, args.Host, params)
		} else {
			result, err = modules.Hostcachedimages.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Hostcachedimages.GetColumns(s))
		return nil
	})

	type HostCachedImageUpdateOptions struct {
		HOST   string `help:"ID or Name of Host"`
		IMAGE  string `help:"ID or name of image"`
		Status string `help:"Status"`
	}
	R(&HostCachedImageUpdateOptions{}, "host-cachedimage-update", "Update host cached image", func(s *mcclient.ClientSession, args *HostCachedImageUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Status) > 0 {
			params.Add(jsonutils.NewString(args.Status), "status")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result, err := modules.Hostcachedimages.Update(s, args.HOST, args.IMAGE, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
