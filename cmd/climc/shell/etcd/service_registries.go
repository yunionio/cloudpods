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

package etcd

import (
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/etcd"
)

func init() {
	type ServiceRegistryListOptions struct {
	}
	shell.R(&ServiceRegistryListOptions{}, "service-registry-list", "List all service registries", func(s *mcclient.ClientSession, args *ServiceRegistryListOptions) error {
		results, err := etcd.ServiceRegistryManager.List(s, nil)
		if err != nil {
			return err
		}
		printutils.PrintJSONList(results, etcd.ServiceRegistryManager.GetColumns(s))
		return nil
	})
}
