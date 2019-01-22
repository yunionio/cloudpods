package shell

import (
	"fmt"
	"sort"
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		ImageType string `help:"image type" choices:"customized|system|shared|market"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *azure.SRegion, args *ImageListOptions) error {
		if images, err := cli.GetImages(args.ImageType); err != nil {
			return err
		} else {
			printList(images, len(images), 0, 0, []string{})
			return nil
		}
	})

	type ImagePublishersOptions struct {
	}
	shellutils.R(&ImagePublishersOptions{}, "image-publisher-list", "List image providers", func(cli *azure.SRegion, args *ImagePublishersOptions) error {
		providers, err := cli.GetImagePublishers(nil)
		if err != nil {
			return err
		}
		sort.Strings(providers)
		fmt.Println(providers)
		return nil
	})

	type ImageOfferedIDOptions struct {
		Publisher []string `help:"publisher candidates"`
		Offer     []string `help:"offer candidates"`
		Sku       []string `help:"sku candidates"`
		Version   []string `help:"version candidates"`
		Latest    bool     `help:"show latest version only"`
	}
	shellutils.R(&ImageOfferedIDOptions{}, "public-image-id-list", "List image providers", func(cli *azure.SRegion, args *ImageOfferedIDOptions) error {
		idList, err := cli.GetOfferedImageIDs(args.Publisher, args.Offer, args.Sku, args.Version, args.Latest)
		if err != nil {
			return err
		}
		sort.Strings(idList)
		for _, id := range idList {
			fmt.Println(id)
		}
		return nil
	})

	type ImageCreateOptions struct {
		NAME     string `helo:"Image name"`
		OSTYPE   string `helo:"Operation system" choices:"Linux|Windows"`
		Snapshot string `help:"Snapshot ID"`
		BlobUri  string `helo:"page blob uri"`
		DiskSize int32  `helo:"Image size"`
		Desc     string `help:"Image desc"`
	}

	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *azure.SRegion, args *ImageCreateOptions) error {
		if len(args.Snapshot) > 0 {
			if image, err := cli.CreateImage(args.Snapshot, args.NAME, args.OSTYPE, args.Desc); err != nil {
				return err
			} else {
				printObject(image)
				return nil
			}
		} else {
			if image, err := cli.CreateImageByBlob(args.NAME, args.OSTYPE, args.BlobUri, args.DiskSize); err != nil {
				return err
			} else {
				printObject(image)
				return nil
			}
		}
	})

	type ImageDeleteOptions struct {
		ID string `helo:"Image ID"`
	}

	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *azure.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})

}
