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

	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type ImageListOptions struct {
		ImageId string
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *zstack.SRegion, args *ImageListOptions) error {
		images, err := cli.GetImages(args.ImageId)
		if err != nil {
			return err
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

	type ImageCreateOptions struct {
		FILE     string
		FORMAT   string `choices:"qcow2|raw|iso"`
		PLATFORM string `choices:"Linux|Windows|Other"`
		Desc     string
	}

	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *zstack.SRegion, args *ImageCreateOptions) error {
		f, err := os.Open(args.FILE)
		if err != nil {
			return err
		}
		defer f.Close()
		finfo, err := f.Stat()
		if err != nil {
			return err
		}
		image, err := cli.CreateImage(args.FILE, args.FORMAT, args.PLATFORM, args.Desc, f, finfo.Size())
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

}
