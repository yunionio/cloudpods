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
	"fmt"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type VersionOptions struct {
		SERVICE string `help:"Service type"`
	}
	R(&VersionOptions{}, "version-show", "query backend service for its version", func(s *mcclient.ClientSession, args *VersionOptions) error {
		body, err := modules.GetVersion(s, args.SERVICE)
		if err != nil {
			return err
		}
		fmt.Println(body)
		return nil
	})

	R(&options.VersionListOptions{}, "yunionagent-version-list", "show versions of backend services", func(s *mcclient.ClientSession, opts *options.VersionListOptions) error {
		if len(opts.Region) == 0 {
			opts.Region = s.GetRegion()
		}
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Version.List(s, params)
		if err != nil {
			return err
		}
		printList(result, []string{})
		return nil
	})

	R(&options.VersionGetOptions{}, "yunionagent-version-show", "Show service version", func(s *mcclient.ClientSession, opts *options.VersionGetOptions) error {
		result, err := modules.Version.Get(s, opts.Service, nil)
		if err != nil {
			return err
		}
		ver, err := result.GetString()
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", ver)
		return nil
	})
}
