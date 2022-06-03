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

package misc

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type AppOptionsOptions struct {
		SERVICE string `help:"Service type"`
	}
	R(&AppOptionsOptions{}, "app-options-show", "query backend service for its options", func(s *mcclient.ClientSession, args *AppOptionsOptions) error {
		opts, err := modules.GetAppOptions(s, args.SERVICE)
		if err != nil {
			return err
		}
		printObject(opts)
		return nil
	})
}
