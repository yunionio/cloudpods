package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type TenantListOptions struct {
	}
	R(&TenantListOptions{}, "tenant-list", "List tenants", func(s *mcclient.ClientSession, args *TenantListOptions) error {
		result, err := modules.Tenants.List(s, nil)
		if err != nil {
			return err
		}
		printList(result, modules.Tenants.GetColumns(s))
		return nil
	})
}
