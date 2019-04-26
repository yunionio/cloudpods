package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type ClusterListOptions struct {
		StorageId string
		DiskId    string
	}
	shellutils.R(&ClusterListOptions{}, "cluster-list", "List clusters", func(cli *zstack.SRegion, args *ClusterListOptions) error {
		clusters, err := cli.GetClusters()
		if err != nil {
			return err
		}
		printList(clusters, len(clusters), 0, 0, []string{})
		return nil
	})
}
