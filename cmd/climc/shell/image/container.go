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

package image

import (
	"io"
	"os"
	"strings"

	"github.com/cheggaaa/pb/v3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/image"
)

func init() {
	cmd := shell.NewResourceCmd(modules.ContainerRegistries)
	cmd.List(new(options.ContainerRegistryListOptions))
	cmd.Show(new(options.ContainerRegistryIdOptions))
	cmd.Create(new(options.ContainerRegistryCreateOptions))
	cmd.Delete(new(options.ContainerRegistryIdOptions))
	cmd.Get("images", new(options.ContainerRegistryGetImagesOptions))
	cmd.Get("image-tags", new(options.ContainerRegistryGetImageTagsOptions))
	cmd.Get("config", new(options.ContainerRegistryIdOptions))
	cmd.PerformClass("import-from-kubeserver", new(options.ContainerRegistryImportOptions))
	cmd.Perform("public", &baseoptions.BasePublicOptions{})
	cmd.Perform("private", &baseoptions.BaseIdOptions{})

	type UploadOptions struct {
		REGISTRY string `help:"The name or id of registry" json:"-"`
		FILE     string `help:"The container tar image" json:"-"`
		Name     string `help:"Override image name" json:"name"`
		Tag      string `help:"Override image tag" json:"tag"`
	}
	R(new(UploadOptions), "container-registry-upload-image", "Upload a docker image", func(s *mcclient.ClientSession, args *UploadOptions) error {
		f, err := os.Open(args.FILE)
		if err != nil {
			return err
		}
		defer f.Close()
		finfo, err := f.Stat()
		if err != nil {
			return err
		}
		size := finfo.Size()
		bar := pb.Full.Start64(size)
		barReader := bar.NewProxyReader(f)
		param := jsonutils.Marshal(args)
		img, err := modules.ContainerRegistries.UploadImage(s, args.REGISTRY, param, barReader, size)
		if err != nil {
			return err
		}
		printObject(img)
		return nil
	})

	type DownloadOptions struct {
		NAME     string `help:"The name of image, e.g. 'influxdb:1.7.7'"`
		Registry string `help:"The name or id of registry" json:"-"`
		Output   string `help:"Saved file path"`
		Insecure bool   `help:"Set insecure"`
		Username string `help:"Image registry username"`
		Password string `help:"Image registry password"`
	}
	R(new(DownloadOptions), "container-registry-download-image", "Download container image to a file", func(s *mcclient.ClientSession, args *DownloadOptions) error {
		var (
			fileName string
			src      io.Reader
			size     int64
			err      error
		)
		if args.Registry != "" {
			parts := strings.Split(args.NAME, ":")
			if len(parts) != 2 {
				return errors.Errorf("invalid NAME %q, use format <name>:<tag>", args.NAME)
			}
			fileName, src, size, err = modules.ContainerRegistries.DownloadImage(s, args.Registry, parts[0], parts[1])
			if err != nil {
				return errors.Wrap(err, "download image")
			}
		} else {
			fileName, src, size, err = modules.ContainerRegistries.DownloadImageByManager(s, &modules.DownloadImageByManagerInput{
				Insecure: args.Insecure,
				Image:    args.NAME,
				Username: args.Username,
				Password: args.Password,
			})
			if err != nil {
				return errors.Wrap(err, "download image by manager")
			}
		}
		output := args.Output
		if output == "" && fileName != "" {
			output = fileName
		}
		if output == "" {
			return errors.Errorf("--output filepath must provide")
		}
		f, err := os.Create(output)
		if err != nil {
			return errors.Wrapf(err, "create saved file: %q", args.Output)
		}
		defer f.Close()
		bar := pb.Full.Start64(size)
		barReader := bar.NewProxyReader(src)
		if _, err := io.Copy(f, barReader); err != nil {
			return errors.Wrap(err, "save image")
		}
		return nil
	})

	imgCmd := shell.NewResourceCmd(&modules.ContainerImages)
	imgCmd.List(new(options.ContainerImageListOptions))
	imgCmd.Show(new(options.ContainerImageIdOptions))
	imgCmd.Create(new(options.ContainerImageCreateOptions))
	imgCmd.Update(new(options.ContainerImageUpdateOptions))
	imgCmd.Delete(new(options.ContainerImageIdOptions))
	imgCmd.Perform("public", &baseoptions.BasePublicOptions{})
	imgCmd.Perform("private", &baseoptions.BaseIdOptions{})
}
