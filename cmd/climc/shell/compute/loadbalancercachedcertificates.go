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
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	R(&options.LoadbalancerCertificateGetOptions{}, "lbcert-cache-show", "Show cached lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateGetOptions) error {
		lbcert, err := modules.LoadbalancerCachedCertificates.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})

	R(&options.LoadbalancerCachedCertificateListOptions{}, "lbcert-cache-list", "List cached lbcerts", func(s *mcclient.ClientSession, opts *options.LoadbalancerCachedCertificateListOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerCachedCertificates.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerCachedCertificates.GetColumns(s))
		return nil
	})

	R(&options.LoadbalancerCachedCertificateCreateOptions{}, "lbcert-cache-create", "Create cached lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCachedCertificateCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}

		lbcert, err := modules.LoadbalancerCachedCertificates.Create(s, params)
		if err != nil {
			return err
		}

		printObject(lbcert)
		return nil
	})

	R(&options.LoadbalancerCertificateDeleteOptions{}, "lbcert-cache-delete", "Delete cached lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateDeleteOptions) error {
		lbcert, err := modules.LoadbalancerCachedCertificates.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificateDeleteOptions{}, "lbcert-cache-purge", "Purge cached lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateDeleteOptions) error {
		lbcert, err := modules.LoadbalancerCachedCertificates.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
}
