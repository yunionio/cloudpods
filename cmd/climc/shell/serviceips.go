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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ServiceIpListOptions struct {
		Limit       int64  `help:"Limit, default 0, i.e. no limit" default:"20"`
		Offset      int64  `help:"Offset, default 0, i.e. no offset"`
		ServiceType string `help:"Filter by type"`
		ServiceId   string `help:"Filter by Service Id"`
	}
	R(&ServiceIpListOptions{}, "serviceip-list", "List serviceip", func(s *mcclient.ClientSession, args *ServiceIpListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		serviceips, err := modules.ServiceIp.List(s, params)
		if err != nil {
			return err
		}
		printList(serviceips, modules.ServiceIp.GetColumns(s))
		return nil
	})
}
