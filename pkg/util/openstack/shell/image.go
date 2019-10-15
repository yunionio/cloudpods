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
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Name   string
		Id     string
		Status string
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *openstack.SRegion, args *ImageListOptions) error {
		images, err := cli.GetImages(args.Name, args.Status, args.Id)
		if err != nil {
			return err
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

	type ImageOptions struct {
		ID string
	}

	shellutils.R(&ImageOptions{}, "image-show", "Show image", func(cli *openstack.SRegion, args *ImageOptions) error {
		image, err := cli.GetImages("", "", args.ID)
		if err != nil {
			return err
		}
		printObject(image[0])
		return nil
	})

	shellutils.R(&ImageOptions{}, "image-delete", "Delete image", func(cli *openstack.SRegion, args *ImageOptions) error {
		return cli.DeleteImage(args.ID)
	})

	type ImageCreateOptions struct {
		NAME          string
		OsType        string `help:"os type" default:"linux" choices:"linux|windows"`
		OsDistro      string
		MinDiskSizeGB int
		MinRamMb      int
	}

	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *openstack.SRegion, args *ImageCreateOptions) error {
		image, err := cli.CreateImage(args.NAME, args.OsType, args.OsDistro, args.MinDiskSizeGB, args.MinRamMb)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

}
