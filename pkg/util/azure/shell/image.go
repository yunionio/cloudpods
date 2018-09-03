package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List intances", func(cli *azure.SRegion, args *ImageListOptions) error {
		if images, err := cli.GetImages(); err != nil {
			return err
		} else {
			printList(images, len(images), args.Offset, args.Limit, []string{})
			return nil
		}
	})
}
