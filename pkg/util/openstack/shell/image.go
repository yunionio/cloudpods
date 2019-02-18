package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Name   string
		Ids    []string
		Status string
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *openstack.SRegion, args *ImageListOptions) error {
		images, err := cli.GetImages(args.Name, args.Status, args.Ids)
		if err != nil {
			return err
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

	type ImageOptions struct {
		ID string
	}

	shellutils.R(&ImageOptions{}, "image-show", "Show image", func(cli *openstack.SRegion, args *ImageOptions) error {
		image, err := cli.GetImages("", "", []string{args.ID})
		if err != nil {
			return err
		}
		printObject(image[0])
		return nil
	})

	shellutils.R(&ImageOptions{}, "image-delete", "Delete image", func(cli *openstack.SRegion, args *ImageOptions) error {
		return cli.DeleteImage(args.ID)
	})

	type ImageCreateOptions struct {
		NAME string
	}

	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *openstack.SRegion, args *ImageCreateOptions) error {
		image, err := cli.CreateImage(args.NAME)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

}
