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

package yunionconf

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
)

func init() {
	type BugReportStatusOptions struct {
	}

	R(&BugReportStatusOptions{}, "bug-report-status", "Show bug report status", func(s *mcclient.ClientSession, args *BugReportStatusOptions) error {
		ret, err := yunionconf.BugReport.GetBugReportEnabled(s, nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	type BugReportEnableOptions struct {
	}

	R(&BugReportEnableOptions{}, "bug-report-enable", "Enable bug report", func(s *mcclient.ClientSession, args *BugReportEnableOptions) error {
		ret, err := yunionconf.BugReport.DoBugReportEnable(s, nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&BugReportEnableOptions{}, "bug-report-disable", "Disable bug report", func(s *mcclient.ClientSession, args *BugReportEnableOptions) error {
		ret, err := yunionconf.BugReport.DoBugReportDisable(s, nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

}
