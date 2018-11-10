package shell

import (
	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Status string   `help:"image status type" choices:"Creating|Available|UnAvailable|CreateFailed"`
		Owner  string   `help:"Owner type" choices:"amazon|self|microsoft|aws-marketplace"`
		Id     []string `help:"Image ID"`
		Name   string   `help:"image name"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *aws.SRegion, args *ImageListOptions) error {
		images, total, e := cli.GetImages(aws.ImageStatusType(args.Status), aws.ImageOwnerType(args.Owner), args.Id, args.Name, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(images, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `help:"ID or Name to delete"`
	}
	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *aws.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})
}
