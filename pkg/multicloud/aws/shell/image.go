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
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Status             string   `help:"image status type" choices:"Creating|Available|UnAvailable|CreateFailed"`
		Owner              string   `help:"Owner type, e.g. self, system or all" choices:"self|system|all"`
		VirtualizationType string   `help:"virtualization type" choices:"hvm|paravirtual"`
		Id                 []string `help:"Image ID"`
		Name               string   `help:"image name"`
		RawOwner           []string `help:"raw owner id"`
		VolumeType         string   `help:"image volume type" choices:"gp2|io1|st1|sc1|standard"`
		Latest             bool     `help:"show latest image only"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *aws.SRegion, args *ImageListOptions) error {
		var owners []aws.TImageOwnerType
		switch args.Owner {
		case "self":
			owners = aws.ImageOwnerSelf
		case "system":
			owners = aws.ImageOwnerSystem
		}
		images, e := cli.GetImages(args.Status, owners, args.Id, args.Name, args.VirtualizationType, args.RawOwner, args.VolumeType, args.Latest)
		if e != nil {
			return e
		}
		printList(images, 0, 0, 0, []string{})
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `help:"ID or Name to delete"`
	}
	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *aws.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})
}
