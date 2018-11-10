package shell

import (
	"yunion.io/x/onecloud/pkg/util/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Status string `help:"Image status"`
		Owner  string `help:"Image owner" choices:"PRIVATE_IMAGE|PUBLIC_IMAGE|MARKET_IMAGE|SHARED_IMAGE"`
		Image  string `help:"Image Id"`
		Name   string `help:"Image Name"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *qcloud.SRegion, args *ImageListOptions) error {
		imageIds := []string{}
		if len(args.Image) > 0 {
			imageIds = append(imageIds, args.Image)
		}
		images, total, err := cli.GetImages(args.Status, args.Owner, imageIds, args.Name, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(images, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type ImageCreateOptions struct {
		NAME      string `helo:"Image name"`
		OSTYPE    string `helo:"Operation system" choices:"CentOS|Ubuntu|Debian|OpenSUSE|SUSE|CoreOS|FreeBSD|Other Linux|Windows Server 2008|Windows Server 2012|Windows Server 2016"`
		OSARCH    string `help:"OS Architecture" choices:"x86_64|i386"`
		osVersion string `help:"OS Version"`
		URL       string `helo:"Cos URL"`
	}

	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *qcloud.SRegion, args *ImageCreateOptions) error {
		image, err := cli.ImportImage(args.NAME, args.OSARCH, args.OSTYPE, args.osVersion, args.URL)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `helo:"Image ID"`
	}

	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *qcloud.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})

}
