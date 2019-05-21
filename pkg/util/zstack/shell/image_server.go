package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type ImageServerListOptions struct {
		ZoneId string
	}
	shellutils.R(&ImageServerListOptions{}, "image-server-list", "List image servers", func(cli *zstack.SRegion, args *ImageServerListOptions) error {
		servers, err := cli.GetImageServers(args.ZoneId)
		if err != nil {
			return err
		}
		printList(servers, 0, 0, 0, []string{})
		return nil
	})

}
