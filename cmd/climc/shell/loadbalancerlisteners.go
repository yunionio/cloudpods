// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func printLbBackendStatus(backendStatus jsonutils.JSONObject) error {
	arr, ok := backendStatus.(*jsonutils.JSONArray)
	if !ok {
		return fmt.Errorf("want json array, got %s", backendStatus.String())
	}
	objList, err := arr.GetArray()
	if err != nil {
		return err
	}
	listResult := &modulebase.ListResult{
		Data: objList,
	}
	columns := []string{
		"id",
		"name",
		"backend_type",
		"backend_id",
		"address",
		"port",
		"weight",
		"check_time",
		"check_status",
		"check_code",
	}
	printList(listResult, columns)
	return nil
}

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
	R(&options.LoadbalancerListenerGetBackendStatusOptions{}, "lblistener-backend-status", "Get lblistene backend status", func(s *mcclient.ClientSession, opts *options.LoadbalancerListenerGetBackendStatusOptions) error {
		backendStatus, err := modules.LoadbalancerListeners.GetSpecific(s, opts.ID, "backend-status", nil)
		if err != nil {
			return err
		}
		return printLbBackendStatus(backendStatus)
	})
}
