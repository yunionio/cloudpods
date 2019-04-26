package shell

import (
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

}
