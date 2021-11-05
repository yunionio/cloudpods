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

package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.DNSListOptions{}, "dns-list", "List dns records", func(s *mcclient.ClientSession, opts *options.DNSListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.DNSRecords.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DNSRecords.GetColumns(s))
		return nil
	})

	R(&options.DNSCreateOptions{}, "dns-create", "Create dns record", func(s *mcclient.ClientSession, opts *options.DNSCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		rec, e := modules.DNSRecords.Create(s, params)
		if e != nil {
			return e
		}
		printObject(rec)
		return nil
	})

	R(&options.DNSGetOptions{}, "dns-show", "Show details of a dns records", func(s *mcclient.ClientSession, opts *options.DNSGetOptions) error {
		dns, e := modules.DNSRecords.Get(s, opts.ID, nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&options.DNSUpdateOptions{}, "dns-update", "Update details of a dns records", func(s *mcclient.ClientSession, opts *options.DNSUpdateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		dns, e := modules.DNSRecords.Update(s, opts.ID, params)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&options.DNSGetOptions{}, "dns-delete", "Delete a dns record", func(s *mcclient.ClientSession, opts *options.DNSGetOptions) error {
		dns, e := modules.DNSRecords.Delete(s, opts.ID, nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&options.DNSGetOptions{}, "dns-public", "Make a dns record publicly available", func(s *mcclient.ClientSession, opts *options.DNSGetOptions) error {
		dns, e := modules.DNSRecords.PerformAction(s, opts.ID, "public", nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&options.DNSGetOptions{}, "dns-private", "Make a dns record private", func(s *mcclient.ClientSession, opts *options.DNSGetOptions) error {
		dns, e := modules.DNSRecords.PerformAction(s, opts.ID, "private", nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&options.DNSGetOptions{}, "dns-enable", "Enable dns record", func(s *mcclient.ClientSession, opts *options.DNSGetOptions) error {
		dns, e := modules.DNSRecords.PerformAction(s, opts.ID, "enable", nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&options.DNSGetOptions{}, "dns-disable", "Disable dns record", func(s *mcclient.ClientSession, opts *options.DNSGetOptions) error {
		dns, e := modules.DNSRecords.PerformAction(s, opts.ID, "disable", nil)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&options.DNSUpdateRecordsOptions{}, "dns-add-records", "Add DNS records to a name", func(s *mcclient.ClientSession, opts *options.DNSUpdateRecordsOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		dns, e := modules.DNSRecords.PerformAction(s, opts.ID, "add-records", params)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

	R(&options.DNSUpdateRecordsOptions{}, "dns-remove-records", "Remove DNS records from a name", func(s *mcclient.ClientSession, opts *options.DNSUpdateRecordsOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		dns, e := modules.DNSRecords.PerformAction(s, opts.ID, "remove-records", params)
		if e != nil {
			return e
		}
		printObject(dns)
		return nil
	})

}
