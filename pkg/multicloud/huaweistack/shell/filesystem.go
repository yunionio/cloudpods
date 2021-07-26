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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	huawei "yunion.io/x/onecloud/pkg/multicloud/huaweistack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type FileSystemListOptions struct {
	}
	shellutils.R(&FileSystemListOptions{}, "file-system-list", "List FileSystem", func(cli *huawei.SRegion, args *FileSystemListOptions) error {
		sfs, err := cli.GetSfsTurbos()
		if err != nil {
			return err
		}
		printList(sfs, 0, 0, 0, []string{})
		return nil
	})

	type FileSystemIdOptions struct {
		ID string `help:"File System ID"`
	}
	shellutils.R(&FileSystemIdOptions{}, "file-system-delete", "Delete filesystem", func(cli *huawei.SRegion, args *FileSystemIdOptions) error {
		return cli.DeleteSfsTurbo(args.ID)
	})

	shellutils.R(&FileSystemIdOptions{}, "file-system-show", "Show filesystem", func(cli *huawei.SRegion, args *FileSystemIdOptions) error {
		fs, err := cli.GetSfsTurbo(args.ID)
		if err != nil {
			return err
		}
		printObject(fs)
		return nil
	})

	shellutils.R(&cloudprovider.FileSystemCraeteOptions{}, "file-system-create", "Create filesystem", func(cli *huawei.SRegion, args *cloudprovider.FileSystemCraeteOptions) error {
		fs, err := cli.CreateSfsTurbo(args)
		if err != nil {
			return err
		}
		printObject(fs)
		return nil
	})

}
