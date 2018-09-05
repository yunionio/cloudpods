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
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *azure.SRegion, args *ImageListOptions) error {
		if images, err := cli.GetImages(); err != nil {
			return err
		} else {
			printList(images, len(images), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type ImageCreateOptions struct {
		Name    string `helo:"Image name"`
		OsType  string `helo:"Operation system" choices:"Linux|Windows"`
		BlobURI string `helo:"page blob uri"`
	}

	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *azure.SRegion, args *ImageCreateOptions) error {
		if image, err := cli.CreateImageByBlob(args.Name, args.OsType, args.BlobURI); err != nil {
			return err
		} else {
			printObject(image)
			return nil
		}
	})
}
