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

type cliTask struct {
	cli     *Client
	l       IServiceCatalogChangeListener
	catalog IServiceCatalog

	taskType string
}

func (t *cliTask) Run() {
	switch t.taskType {
	case "GetServiceCatalog":
		t.l.OnServiceCatalogChange(t.cli.GetServiceCatalog())
	case "OnServiceCatalogChange":
		for i := range t.cli.catalogListeners {
			t.cli.catalogListeners[i].OnServiceCatalogChange(t.catalog)
		}
	}
}

func (t *cliTask) Dump() string {
	return ""
}

func (cli *Client) RegisterCatalogListener(l IServiceCatalogChangeListener) {
	cli.catalogListeners = append(cli.catalogListeners, l)
	task := &cliTask{
		cli:      cli,
		l:        l,
		taskType: "GetServiceCatalog",
	}
	if cli.GetServiceCatalog() != nil {
		listenerWorker.Run(task, nil, nil)
	}
}

func (cli *Client) SetServiceCatalog(catalog IServiceCatalog) {
	cli._serviceCatalog = catalog
	task := &cliTask{
		cli:      cli,
		catalog:  catalog,
		taskType: "OnServiceCatalogChange",
	}

	listenerWorker.Run(task, nil, nil)
}

func (this *Client) GetServiceCatalog() IServiceCatalog {
	return this._serviceCatalog
}
