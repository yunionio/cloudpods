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
	"os"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/nutanix"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
	}
	shellutils.R(&ImageListOptions{}, "image-list", "list hosts", func(cli *nutanix.SRegion, args *ImageListOptions) error {
		images, err := cli.GetImages()
		if err != nil {
			return err
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

	type ImageIdOptions struct {
		ID string
	}

	shellutils.R(&ImageIdOptions{}, "image-show", "show host", func(cli *nutanix.SRegion, args *ImageIdOptions) error {
		image, err := cli.GetImage(args.ID)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

	type ImageUploadOptions struct {
		STOREG_ID string
		NAME      string
		FILE      string
	}

	shellutils.R(&ImageUploadOptions{}, "image-upload", "upload host", func(cli *nutanix.SRegion, args *ImageUploadOptions) error {
		fi, err := os.Open(args.FILE)
		if err != nil {
			return err
		}
		defer fi.Close()

		stat, _ := fi.Stat()
		image, err := cli.CreateImage(
			args.STOREG_ID,
			&cloudprovider.SImageCreateOption{
				ImageName: args.NAME,
			},
			stat.Size(),
			fi,
			nil,
		)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

}
