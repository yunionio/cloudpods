package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.LoadbalancerBackendCreateOptions{}, "lbbackend-create", "Create lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendCreateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lbbackend, err := modules.LoadbalancerBackends.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
	R(&options.LoadbalancerBackendGetOptions{}, "lbbackend-show", "Show lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendGetOptions) error {
		lbbackend, err := modules.LoadbalancerBackends.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
	R(&options.LoadbalancerBackendListOptions{}, "lbbackend-list", "List lbbackends", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerBackends.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerBackends.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerBackendUpdateOptions{}, "lbbackend-update", "Update lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendUpdateOptions) error {
		params, err := options.StructToParams(opts)
		lbbackend, err := modules.LoadbalancerBackends.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
	R(&options.LoadbalancerBackendDeleteOptions{}, "lbbackend-delete", "Delete lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendDeleteOptions) error {
		lbbackend, err := modules.LoadbalancerBackends.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
	R(&options.LoadbalancerBackendDeleteOptions{}, "lbbackend-purge", "Purge lbbackend", func(s *mcclient.ClientSession, opts *options.LoadbalancerBackendDeleteOptions) error {
		lbbackend, err := modules.LoadbalancerBackends.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(lbbackend)
		return nil
	})
}
