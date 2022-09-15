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
	"time"

	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type LoganalyticsWorkspaceListOptions struct {
	}
	shellutils.R(&LoganalyticsWorkspaceListOptions{}, "loganalytics-workspace-list", "List loganalytics workspaces", func(cli *azure.SRegion, args *LoganalyticsWorkspaceListOptions) error {
		workspaces, err := cli.GetClient().GetLoganalyticsWorkspaces()
		if err != nil {
			return err
		}
		printList(workspaces, len(workspaces), 0, 0, []string{})
		return nil
	})

	type DiskUsageListOptions struct {
		WORKSPACE_ID string
		NAME         string
		START        time.Time
		END          time.Time
	}

	shellutils.R(&DiskUsageListOptions{}, "disk-usage-list", "List disk usage", func(cli *azure.SRegion, args *DiskUsageListOptions) error {
		usage, err := cli.GetClient().GetInstanceDiskUsage(args.WORKSPACE_ID, args.NAME, args.START, args.END)
		if err != nil {
			return err
		}
		printList(usage, len(usage), 0, 0, []string{})
		return nil
	})

}
