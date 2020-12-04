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
	"sort"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		ImageType string `help:"image type" choices:"customized|system|shared|market"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *azure.SRegion, args *ImageListOptions) error {
		images, err := cli.GetImages(cloudprovider.TImageType(args.ImageType))
		if err != nil {
			return err
		}
		printList(images, len(images), 0, 0, []string{})
		return nil
	})

	type ImagePublishersOptions struct {
	}
	shellutils.R(&ImagePublishersOptions{}, "image-publisher-list", "List image providers", func(cli *azure.SRegion, args *ImagePublishersOptions) error {
		providers, err := cli.GetImagePublishers(nil)
		if err != nil {
			return err
		}
		sort.Strings(providers)
		fmt.Println(providers)
		return nil
	})

	type ImageOfferedIDOptions struct {
		Publisher []string `help:"publisher candidates"`
		Offer     []string `help:"offer candidates"`
		Sku       []string `help:"sku candidates"`
		Version   []string `help:"version candidates"`
		Latest    bool     `help:"show latest version only"`
	}
	shellutils.R(&ImageOfferedIDOptions{}, "public-image-id-list", "List image providers", func(cli *azure.SRegion, args *ImageOfferedIDOptions) error {
		idList, err := cli.GetOfferedImageIDs(args.Publisher, args.Offer, args.Sku, args.Version, args.Latest)
		if err != nil {
			return err
		}
		//sort.Strings(idList)
		for id, image := range idList {
			fmt.Printf("id: %s detail: %v\n", id, image)
		}
		return nil
	})

	type ImageCreateOptions struct {
		NAME     string `helo:"Image name"`
		OSTYPE   string `helo:"Operation system" choices:"Linux|Windows"`
		Snapshot string `help:"Snapshot ID"`
		BlobUri  string `helo:"page blob uri"`
		DiskSize int32  `helo:"Image size"`
		Desc     string `help:"Image desc"`
	}

	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *azure.SRegion, args *ImageCreateOptions) error {
		if len(args.Snapshot) > 0 {
			if image, err := cli.CreateImage(args.Snapshot, args.NAME, args.OSTYPE, args.Desc); err != nil {
				return err
			} else {
				printObject(image)
				return nil
			}
		} else {
			if image, err := cli.CreateImageByBlob(args.NAME, args.OSTYPE, args.BlobUri, args.DiskSize); err != nil {
				return err
			} else {
				printObject(image)
				return nil
			}
		}
	})

	type ImageIdOptions struct {
		ID string `helo:"Image ID"`
	}

	shellutils.R(&ImageIdOptions{}, "image-delete", "Delete image", func(cli *azure.SRegion, args *ImageIdOptions) error {
		return cli.DeleteImage(args.ID)
	})

	shellutils.R(&ImageIdOptions{}, "image-show", "Delete image", func(cli *azure.SRegion, args *ImageIdOptions) error {
		image, err := cli.GetImageById(args.ID)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

}
