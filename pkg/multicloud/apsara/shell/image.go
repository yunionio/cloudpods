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

	"yunion.io/x/onecloud/pkg/multicloud/apsara"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Status string   `help:"image status type" choices:"Creating|Available|UnAvailable|CreateFailed"`
		Owner  string   `help:"Owner type" choices:"system|self|others|marketplace"`
		Id     []string `help:"Image ID"`
		Name   string   `help:"image name"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *apsara.SRegion, args *ImageListOptions) error {
		images, total, e := cli.GetImages(apsara.ImageStatusType(args.Status), apsara.ImageOwnerType(args.Owner), args.Id, args.Name, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(images, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type ImageShowOptions struct {
		ID string `help:"image ID"`
	}
	shellutils.R(&ImageShowOptions{}, "image-show", "Show image", func(cli *apsara.SRegion, args *ImageShowOptions) error {
		img, err := cli.GetImage(args.ID)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `help:"ID or Name to delete"`
	}
	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *apsara.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})

	type ImageCreateOptions struct {
		SNAPSHOT string `help:"Snapshot id"`
		NAME     string `help:"Image name"`
		Desc     string `help:"Image desc"`
	}
	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *apsara.SRegion, args *ImageCreateOptions) error {
		imageId, err := cli.CreateImage(args.SNAPSHOT, args.NAME, args.Desc)
		if err != nil {
			return err
		}
		fmt.Println(imageId)
		return nil
	})

	type ImageExportOptions struct {
		ID     string `help:"ID or Name to export"`
		BUCKET string `help:"Bucket name"`
	}

	shellutils.R(&ImageExportOptions{}, "image-export", "Export image", func(cli *apsara.SRegion, args *ImageExportOptions) error {
		task, err := cli.ExportImage(args.ID, args.BUCKET)
		if err != nil {
			return err
		}
		printObject(task)
		return nil
	})
}
