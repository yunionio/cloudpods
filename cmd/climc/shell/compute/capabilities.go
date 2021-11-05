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
	type CapabilitiesOptions struct {
		Domain string `help:"ID or name of domain"`
		Scope  string `help:"query scope" choices:"system|domain|project"`
	}
	R(&CapabilitiesOptions{}, "capabilities", "Show backend capabilities", func(s *mcclient.ClientSession, args *CapabilitiesOptions) error {
		query, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Capabilities.List(s, query)
		if err != nil {
			return err
		}
		printObject(result.Data[0])
		return nil
	})
}
