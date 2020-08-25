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

package mcclient

type IServiceCatalog interface {
	Len() int
	GetServiceURL(service, region, zone, endpointType string) (string, error)
	GetServiceURLs(service, region, zone, endpointType string) ([]string, error)
	GetInternalServices(region string) []string
	GetExternalServices(region string) []ExternalService
	GetServicesByInterface(region string, infType string) []ExternalService
}

type IServiceCatalogChangeListener interface {
	OnServiceCatalogChange(catalog IServiceCatalog)
}

func (cli *Client) RegisterCatalogListener(l IServiceCatalogChangeListener) {
	cli.catalogListeners = append(cli.catalogListeners, l)
	if cli.GetServiceCatalog() != nil {
		cli.listenerWorker.Run(func() {
			l.OnServiceCatalogChange(cli.GetServiceCatalog())
		}, nil, nil)
	}
}

func (cli *Client) SetServiceCatalog(catalog IServiceCatalog) {
	cli._serviceCatalog = catalog
	cli.listenerWorker.Run(func() {
		for i := range cli.catalogListeners {
			cli.catalogListeners[i].OnServiceCatalogChange(catalog)
		}
	}, nil, nil)
}

func (this *Client) GetServiceCatalog() IServiceCatalog {
	return this._serviceCatalog
}
