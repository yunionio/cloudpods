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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ServiceCatalogListOpions struct {
		options.BaseListOptions
	}

	R(&ServiceCatalogListOpions{}, "service-catalog-list", "List service catalog", func(s *mcclient.ClientSession,
		opts *ServiceCatalogListOpions) error {

		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		ret, err := modules.ServiceCatalog.List(s, params)
		if err != nil {
			return err
		}
		printList(ret, modules.ServiceCatalog.GetColumns(s))
		return nil
	})

	type ServiceCatalogCreateOptions struct {
		NAME          string `help:"Name of service catalog"`
		GuestTemplate string `help:"guest template of service catalog"`
		IconUrl       string `help:"icon url of service catalog"`
		GenerateName  bool   `help:"whether to generate name"`
	}

	R(&ServiceCatalogCreateOptions{}, "service-catalog-create", "Create a service catalog",
		func(s *mcclient.ClientSession, opts *ServiceCatalogCreateOptions) error {

			params := jsonutils.NewDict()
			if opts.GenerateName {
				params.Add(jsonutils.NewString(opts.NAME), "generate_name")
			} else {
				params.Add(jsonutils.NewString(opts.NAME), "name")
			}
			if len(opts.GuestTemplate) != 0 {
				params.Add(jsonutils.NewString(opts.GuestTemplate), "guest_template")
			}
			if len(opts.IconUrl) != 0 {
				params.Add(jsonutils.NewString(opts.IconUrl), "icon_url")
			}
			serviceCatalog, err := modules.ServiceCatalog.Create(s, params)
			if err != nil {
				return err
			}
			printObject(serviceCatalog)
			return nil
		},
	)

	type ServiceCatalogOptions struct {
		ID string `help:"ID or name of service catalog"`
	}

	R(&ServiceCatalogOptions{}, "service-catalog-show", "show a service catalog", func(s *mcclient.ClientSession,
		opts *ServiceCatalogOptions) error {

		sc, err := modules.ServiceCatalog.Get(s, opts.ID, jsonutils.JSONNull)
		if err != nil {
			return err
		}
		printObject(sc)
		return nil
	})

	R(&ServiceCatalogOptions{}, "service-catalog-delete", "delete a service catalog", func(s *mcclient.ClientSession,
		opts *ServiceCatalogOptions) error {

		sc, err := modules.ServiceCatalog.Delete(s, opts.ID, jsonutils.JSONNull)
		if err != nil {
			return err
		}
		printObject(sc)
		return nil
	})

	type ServiceCatalogUpdateOptions struct {
		ServiceCatalogOptions
		Name          string `help:"Name of service catalog"`
		GuestTemplate string `help:"guest template of service catalog"`
		IconUrl       string `help:"icon url of service catalog"`
	}

	R(&ServiceCatalogUpdateOptions{}, "service-catalog-update", "update a service catalog",
		func(s *mcclient.ClientSession, opts *ServiceCatalogUpdateOptions) error {
			params := jsonutils.NewDict()
			if len(opts.Name) > 0 {
				params.Add(jsonutils.NewString(opts.Name), "name")
			}
			if len(opts.GuestTemplate) > 0 {
				params.Add(jsonutils.NewString(opts.GuestTemplate), "guest_template")
			}
			if len(opts.IconUrl) > 0 {
				params.Add(jsonutils.NewString(opts.IconUrl), "icon_url")
			}
			sc, err := modules.ServiceCatalog.Update(s, opts.ID, params)
			if err != nil {
				return err
			}
			printObject(sc)
			return nil
		},
	)

	type ServiceCatalogDeployOptions struct {
		ServiceCatalogOptions
		Name         string `help:"Name of guest"`
		GenerateName bool   `help:"whether to generate name for guest"`
		Count        int    `help:"count of guest"`
		ProjectID    string `help:"project id of guest"`
	}

	R(&ServiceCatalogDeployOptions{}, "service-catalog-deploy", "deploy", func(s *mcclient.ClientSession,
		opts *ServiceCatalogDeployOptions) error {

		params := jsonutils.NewDict()
		if opts.GenerateName {
			params.Add(jsonutils.NewString(opts.Name), "generate_name")
		} else {
			params.Add(jsonutils.NewString(opts.Name), "name")
		}
		if opts.Count != 0 {
			params.Add(jsonutils.NewInt(int64(opts.Count)), "count")
		}
		if len(opts.ProjectID) > 0 {
			params.Add(jsonutils.NewString(opts.ProjectID), "project_id")
		}

		sc, err := modules.ServiceCatalog.PerformAction(s, opts.ID, "deploy", params)
		if err != nil {
			return err
		}
		printObject(sc)
		return nil
	})

}
