package shell

import (
	"os"

	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type ImageListOptions struct {
		ImageId string
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *zstack.SRegion, args *ImageListOptions) error {
		images, err := cli.GetImages(args.ImageId)
		if err != nil {
			return err
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

	type ImageCreateOptions struct {
		FILE     string
		FORMAT   string `choices:"qcow2|raw|iso"`
		PLATFORM string `choices:"Linux|Windows|Other"`
		Desc     string
	}

	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *zstack.SRegion, args *ImageCreateOptions) error {
		f, err := os.Open(args.FILE)
		if err != nil {
			return err
		}
		defer f.Close()
		finfo, err := f.Stat()
		if err != nil {
			return err
		}
		image, err := cli.CreateImage(args.FILE, args.FORMAT, args.PLATFORM, args.Desc, f, finfo.Size())
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

}
