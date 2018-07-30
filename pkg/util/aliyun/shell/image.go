package shell

import (
	"yunion.io/yunioncloud/pkg/util/aliyun"
)

func init() {
	type ImageListOptions struct {
		Status string   `help:"image status type" choices:"Creating|Available|UnAvailable|CreateFailed"`
		Owner  string   `help:"Owner type" choices:"system|self|others|marketplace"`
		Id     []string `help:"Image ID"`
		Name   string   `help:"image name"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	R(&ImageListOptions{}, "image-list", "List images", func(cli *aliyun.SRegion, args *ImageListOptions) error {
		images, total, e := cli.GetImages(aliyun.ImageStatusType(args.Status), aliyun.ImageOwnerType(args.Owner), args.Id, args.Name, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(images, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `help:"ID or Name to delete"`
	}
	R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *aliyun.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})
}
