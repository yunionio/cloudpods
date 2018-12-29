package etcd

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/etcd"
	"yunion.io/x/onecloud/pkg/util/printutils"
)

func init() {
	type ServiceRegistryListOptions struct {
	}
	shell.R(&ServiceRegistryListOptions{}, "service-registry-list", "List all service registries", func(s *mcclient.ClientSession, args *ServiceRegistryListOptions) error {
		results, err := etcd.ServiceRegistryManager.List(s, nil)
		if err != nil {
			return err
		}
		printutils.PrintJSONList(results, etcd.ServiceRegistryManager.GetColumns(s))
		return nil
	})
}
