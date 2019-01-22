package shell

import (
	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Status string   `help:"image status type" choices:"Creating|Available|UnAvailable|CreateFailed"`
		Owner  []string `help:"Owner type" choices:"amazon|self|microsoft|aws-marketplace"`
		Id     []string `help:"Image ID"`
		Name   string   `help:"image name"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *aws.SRegion, args *ImageListOptions) error {
		var owners []*string
		if len(args.Owner) > 0 {
			owners = make([]*string, 0)
			for i := range args.Owner {
				owners = append(owners, &args.Owner[i])
			}
		}
		images, e := cli.GetImages(aws.ImageStatusType(args.Status), owners, args.Id, args.Name)
		if e != nil {
			return e
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `help:"ID or Name to delete"`
	}
	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *aws.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})
}
