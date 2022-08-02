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
	"yunion.io/x/onecloud/pkg/multicloud/incloudsphere"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageTreeOptions struct {
	}
	shellutils.R(&ImageTreeOptions{}, "image-tree", "list images", func(cli *incloudsphere.SRegion, args *ImageTreeOptions) error {
		trees, err := cli.GetImageTrees()
		if err != nil {
			return err
		}
		for i := range trees {
			printList(trees[i].ToList(), 0, 0, 0, []string{})
		}
		return nil
	})

	type ImageStorageListOptions struct {
		DS_ID string
	}

	shellutils.R(&ImageStorageListOptions{}, "image-storage-list", "list image storages", func(cli *incloudsphere.SRegion, args *ImageStorageListOptions) error {
		storages, err := cli.GetImageStorages(args.DS_ID)
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})

	type ImageListOptions struct {
		STORAGE_ID string
	}

	shellutils.R(&ImageListOptions{}, "image-list", "list images", func(cli *incloudsphere.SRegion, args *ImageListOptions) error {
		images, err := cli.GetImageList(args.STORAGE_ID)
		if err != nil {
			return err
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

}
