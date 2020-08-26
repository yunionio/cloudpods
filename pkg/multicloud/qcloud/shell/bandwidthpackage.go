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
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type BandwidthPackageListOptions struct {
		Ids    []string `help:"Bandwidth package ids"`
		Offset int      `help:"List offset"`
		Limit  int      `help:"List limit"`
	}
	shellutils.R(&BandwidthPackageListOptions{}, "bandwidth-package-list", "List bandwidth packages", func(cli *qcloud.SRegion, args *BandwidthPackageListOptions) error {
		packages, total, err := cli.GetBandwidthPackages(args.Ids, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(packages, total, args.Offset, args.Limit, []string{})
		return nil
	})

}
