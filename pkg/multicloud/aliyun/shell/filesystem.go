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
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type FileSystemListOptions struct {
		Id       string `help:"FileSystem Id"`
		PageSize int    `help:"page size"`
		PageNum  int    `help:"page num"`
	}
	shellutils.R(&FileSystemListOptions{}, "file-system-list", "List FileSystem", func(cli *aliyun.SRegion, args *FileSystemListOptions) error {
		nas, _, err := cli.GetFileSystems(args.Id, args.PageSize, args.PageNum)
		if err != nil {
			return err
		}
		printList(nas, 0, 0, 0, []string{})
		return nil
	})

	type FileSystemDeleteOptions struct {
		ID string `help:"File System ID"`
	}
	shellutils.R(&FileSystemDeleteOptions{}, "file-system-delete", "Delete filesystem", func(cli *aliyun.SRegion, args *FileSystemDeleteOptions) error {
		return cli.DeleteFileSystem(args.ID)
	})

	shellutils.R(&cloudprovider.FileSystemCraeteOptions{}, "file-system-create", "Create filesystem", func(cli *aliyun.SRegion, args *cloudprovider.FileSystemCraeteOptions) error {
		fs, err := cli.CreateFileSystem(args)
		if err != nil {
			return err
		}
		printObject(fs)
		return nil
	})

}
