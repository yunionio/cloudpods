package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.LoadbalancerBackendGroupCreateOptions{}, "lbbackendgroup-create", "Create lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})
	R(&options.LoadbalancerBackendGroupGetOptions{}, "lbbackendgroup-show", "Show lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupGetOptions) error {
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})
	R(&options.LoadbalancerBackendGroupListOptions{}, "lbbackendgroup-list", "List lbbackendgroups", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerBackendGroups.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerBackendGroups.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerBackendGroupUpdateOptions{}, "lbbackendgroup-update", "Update lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupUpdateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})
	R(&options.LoadbalancerBackendGroupDeleteOptions{}, "lbbackendgroup-delete", "Delete lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupDeleteOptions) error {
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})
	R(&options.LoadbalancerBackendGroupDeleteOptions{}, "lbbackendgroup-purge", "Purge lbbackendgroup", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGroupDeleteOptions) error {
		lbbackendgroup, err := modules.LoadbalancerBackendGroups.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(lbbackendgroup)
		return nil
	})
}
