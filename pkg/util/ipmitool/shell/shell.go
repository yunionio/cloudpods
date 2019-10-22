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
	"fmt"

	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type EmptyOptions struct{}

type BootFlagOptions struct {
	FLAG string `help:"Boot flag" choices:"pxe|disk|bios"`
}

type ShutdownOptions struct {
	Soft bool `help:"Do soft shutdown"`
}

func init() {
	shellutils.R(&EmptyOptions{}, "get-sysinfo", "Get system info", func(client ipmitool.IPMIExecutor, _ *EmptyOptions) error {
		info, err := ipmitool.GetSysInfo(client)
		if err != nil {
			return err
		}
		printutils.PrintInterfaceObject(info)
		guid := ipmitool.GetSysGuid(client)
		fmt.Println("UUID:", guid)
		return nil
	})
}
