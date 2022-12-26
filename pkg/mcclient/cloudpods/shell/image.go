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
	"yunion.io/x/pkg/util/shellutils"

	"yunion.io/x/onecloud/pkg/mcclient/cloudpods"
)

func init() {
	type ImageListOptions struct {
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List instances", func(cli *cloudpods.SRegion, args *ImageListOptions) error {
		images, err := cli.GetImages()
		if err != nil {
			return err
		}
		printList(images, 0, 0, 0, nil)
		return nil
	})

	type ImageIdOptions struct {
		ID string
	}

	shellutils.R(&ImageIdOptions{}, "image-show", "Show instance", func(cli *cloudpods.SRegion, args *ImageIdOptions) error {
		image, err := cli.GetImage(args.ID)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

}
