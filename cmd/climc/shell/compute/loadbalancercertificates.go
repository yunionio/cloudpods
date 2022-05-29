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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	R(&options.LoadbalancerCertificateCreateOptions{}, "lbcert-create", "Create lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbcert, err := modules.LoadbalancerCertificates.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificateGetOptions{}, "lbcert-show", "Show lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateGetOptions) error {
		lbcert, err := modules.LoadbalancerCertificates.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificateListOptions{}, "lbcert-list", "List lbcerts", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateListOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerCertificates.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerCertificates.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerCertificateUpdateOptions{}, "lbcert-update", "Update lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateUpdateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbcert, err := modules.LoadbalancerCertificates.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificateDeleteOptions{}, "lbcert-delete", "Delete lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateDeleteOptions) error {
		lbcert, err := modules.LoadbalancerCertificates.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificateDeleteOptions{}, "lbcert-purge", "Purge lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateDeleteOptions) error {
		lbcert, err := modules.LoadbalancerCertificates.PerformAction(s, opts.ID, "purge", nil)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificatePublicOptions{}, "lbcert-public", "Public lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificatePublicOptions) error {
		params := jsonutils.Marshal(opts)
		lbcert, err := modules.LoadbalancerCertificates.PerformAction(s, opts.ID, "public", params)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificatePrivateOptions{}, "lbcert-private", "Private lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificatePrivateOptions) error {
		lbcert, err := modules.LoadbalancerCertificates.PerformAction(s, opts.ID, "private", nil)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
}
