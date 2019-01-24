package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Status string   `help:"image status type" choices:"queued|saving|deleted|killed|active"`
		Owner  string   `help:"Owner type" choices:"gold|private|shared"`
		Id     []string `help:"Image ID"`
		Name   string   `help:"image name"`
		Marker string   `help:"marker"`
		Limit  int      `help:"page Limit"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *huawei.SRegion, args *ImageListOptions) error {
		images, total, e := cli.GetImages(args.Status, huawei.ImageOwnerType(args.Owner), args.Name, args.Limit, args.Marker)
		if e != nil {
			return e
		}
		printList(images, total, 0, args.Limit, []string{})
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `help:"ID or Name to delete"`
	}
	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *huawei.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})
}
