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
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {

	type SystemGetOptions struct {
	}
	shellutils.R(&SystemGetOptions{}, "system-get", "Get details of a system", func(cli redfish.IRedfishDriver, args *SystemGetOptions) error {
		path, sysInfo, err := cli.GetSystemInfo(context.Background())
		if err != nil {
			return err
		}
		fmt.Println(path)
		fmt.Println(jsonutils.Marshal(sysInfo).PrettyString())
		return nil
	})

	shellutils.R(&SystemGetOptions{}, "bios-get", "Get details of a system Bios", func(cli redfish.IRedfishDriver, args *SystemGetOptions) error {
		bios, err := cli.GetBiosInfo(context.Background())
		if err != nil {
			return err
		}
		fmt.Println(jsonutils.Marshal(bios).PrettyString())
		return nil
	})

	type SetNextBootOptions struct {
		DEV string `help:"next boot device"`
	}
	shellutils.R(&SetNextBootOptions{}, "set-next-boot-dev", "Set next boot device", func(cli redfish.IRedfishDriver, args *SetNextBootOptions) error {
		err := cli.SetNextBootDev(context.Background(), args.DEV)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type SystemResetOptions struct {
		ACTION string `help:"reset action"`
	}
	shellutils.R(&SystemResetOptions{}, "system-reset", "Reset system", func(cli redfish.IRedfishDriver, args *SystemResetOptions) error {
		err := cli.Reset(context.Background(), args.ACTION)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

}
