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
	"yunion.io/x/onecloud/pkg/multicloud/ctyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VProjectListOptions struct {
	}
	shellutils.R(&VProjectListOptions{}, "project-list", "List projects", func(cli *ctyun.SRegion, args *VProjectListOptions) error {
		projects, e := cli.FetchProjects()
		if e != nil {
			return e
		}
		printList(projects, 0, 0, 0, nil)
		return nil
	})
}
