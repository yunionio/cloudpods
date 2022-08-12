package compute

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.ModelartsPools)
	cmd.List(&compute.PoolListOptions{})
	// R(&compute.PoolListOptions{}, "pool-list", "List modelarts pool", func(s *mcclient.ClientSession, opts *compute.PoolListOptions) error {
	// 	params, err := options.ListStructToParams(opts)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	result, err := modules.ComputeMetadatas.List(s, params)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	printList(result, []string{})
	// 	return nil
	// })
	cmd.Update(&compute.ElasticSearchUpdateOptions{})
	cmd.Show(&compute.ElasticSearchIdOption{})
	cmd.Delete(&compute.ElasticSearchIdOption{})
	cmd.Perform("syncstatus", &compute.ElasticSearchIdOption{})
	cmd.Get("access-info", &compute.ElasticSearchIdOption{})
}
