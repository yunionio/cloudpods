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

package identity

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.ServiceCertificateCreateOptions{}, "service-cert-create", "Create service cert", func(s *mcclient.ClientSession, opts *options.ServiceCertificateCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		cert, err := modules.ServiceCertificatesV3.Create(s, params)
		if err != nil {
			return err
		}
		printObject(cert)
		return nil
	})
	type ServiceCertificateGetOptions struct {
		ID string `json:"-"`
	}
	R(&ServiceCertificateGetOptions{}, "service-cert-show", "Show service cert", func(s *mcclient.ClientSession, opts *ServiceCertificateGetOptions) error {
		cert, err := modules.ServiceCertificatesV3.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(cert)
		return nil
	})

	type ServiceCertificateListOptions struct {
		options.BaseListOptions
	}
	R(&ServiceCertificateListOptions{}, "service-cert-list", "List service certs", func(s *mcclient.ClientSession, opts *ServiceCertificateListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.ServiceCertificatesV3.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ServiceCertificatesV3.GetColumns(s))
		return nil
	})
	type ServiceCertificateDeleteOptions struct {
		ID string `json:"-"`
	}
	R(&ServiceCertificateDeleteOptions{}, "service-cert-delete", "Delete service cert", func(s *mcclient.ClientSession, opts *ServiceCertificateDeleteOptions) error {
		cert, err := modules.ServiceCertificatesV3.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(cert)
		return nil
	})
}
