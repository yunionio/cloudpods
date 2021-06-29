package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type MemcachedListOptions struct {
		Ids    []string
		Offset int
		Limit  int
	}
	shellutils.R(&MemcachedListOptions{}, "memcached-list", "List memcached", func(cli *qcloud.SRegion, args *MemcachedListOptions) error {
		memcacheds, _, err := cli.GetMemcaches(args.Ids, args.Limit, args.Offset)
		if err != nil {
			return err
		}
		printList(memcacheds, 0, 0, 0, []string{})
		return nil
	})
}
