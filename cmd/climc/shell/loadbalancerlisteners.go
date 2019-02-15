package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	R(&options.LoadbalancerListenerCreateOptions{}, "lblistener-create", "Create lblistener", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerCreateOptions) error {
		// TODO make a generic one
		params := jsonutils.Marshal(opts)
		lblistener, err := modules.LoadbalancerListeners.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lblistener)
		return nil
	})
	R(&options.LoadbalancerListenerGetOptions{}, "lblistener-show", "Show lblistener", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerGetOptions) error {
		lblistener, err := modules.LoadbalancerListeners.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lblistener)
		return nil
	})
	R(&options.LoadbalancerListenerListOptions{}, "lblistener-list", "List lblisteners", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerListeners.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerListeners.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerListenerUpdateOptions{}, "lblistener-update", "Update lblistener", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerUpdateOptions) error {
		params, err := options.StructToParams(opts)
		lblistener, err := modules.LoadbalancerListeners.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lblistener)
		return nil
	})
	R(&options.LoadbalancerListenerDeleteOptions{}, "lblistener-delete", "Delete lblistener", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerDeleteOptions) error {
		lblistener, err := modules.LoadbalancerListeners.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lblistener)
		return nil
	})
	R(&options.LoadbalancerListenerDeleteOptions{}, "lblistener-purge", "Purge lblistener", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerDeleteOptions) error {
		lblistener, err := modules.LoadbalancerListeners.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(lblistener)
		return nil
	})
	R(&options.LoadbalancerListenerActionStatusOptions{}, "lblistener-status", "Change lblistener status", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerActionStatusOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		lblistener, err := modules.LoadbalancerListeners.PerformAction(s, opts.ID, "status", params)
		if err != nil {
			return err
		}
		printObject(lblistener)
		return nil
	})
	R(&options.LoadbalancerListenerActionSyncStatusOptions{}, "lblistener-syncstatus", "Sync lblistener status", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerActionSyncStatusOptions) error {
		lblistener, err := modules.LoadbalancerListeners.PerformAction(s, opts.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(lblistener)
		return nil
	})
}
