package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type TagListOptions struct {
		TagId        string
		ResourceType string
		ResourceId   string
		Tag          string
	}
	shellutils.R(&TagListOptions{}, "system-tag-list", "List system tags", func(cli *zstack.SRegion, args *TagListOptions) error {
		tags, err := cli.GetSysTags(args.TagId, args.ResourceType, args.ResourceId, args.Tag)
		if err != nil {
			return err
		}
		printList(tags, 0, 0, 0, nil)
		return nil
	})
}
