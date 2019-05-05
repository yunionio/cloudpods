package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type ImageListOptions struct {
		Owner string `help:"Owner type" choices:"Base|Business|Custom"`
		Id    string `help:"Image ID"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *ucloud.SRegion, args *ImageListOptions) error {
		images, e := cli.GetImages(args.Owner, args.Id)
		if e != nil {
			return e
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `help:"ID or Name to delete"`
	}
	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *ucloud.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})

	shellutils.R(&ImageDeleteOptions{}, "image-show", "Show image", func(cli *ucloud.SRegion, args *ImageDeleteOptions) error {
		img, err := cli.GetImage(args.ID)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})
}
