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

package k8s

import (
	"io"
	"os"
	"strings"

	"github.com/cheggaaa/pb/v3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initContainerRegistry() {
	cmd := NewK8sResourceCmd(k8s.ContainerRegistries)
	cmd.List(new(o.RegistryListOptions))
	cmd.Show(new(o.RegistryGetOptions))
	cmd.Delete(new(o.RegistryGetOptions))
	cmd.Create(new(o.RegistryCreateOptions))
	cmd.Get("images", new(o.RegistryGetImagesOptions))
	cmd.Get("image-tags", new(o.RegistryGetImageTagsOptions))
	cmd.BatchPerform("public", new(o.RegistryPublicOptions))
	cmd.Perform("private", new(o.RegistryGetOptions))

	type UploadOptions struct {
		REGISTRY string `help:"The name or id of registry" json:"-"`
		FILE     string `help:"The container tar image" json:"-"`
		Name     string `help:"Override image name" json:"name`
		Tag      string `help:"Override image tag" json:"tag"`
	}
	R(new(UploadOptions), "k8s-container-registry-upload-image", "Upload a docker image", func(s *mcclient.ClientSession, args *UploadOptions) error {
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
		img, err := k8s.ContainerRegistries.UploadImage(s, args.REGISTRY, param, barReader, size)
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
		Username string `help:"Image registry username, effective only when --registry is not specified"`
		Password string `help:"Image registry password, effective only when --registry is not specified"`
	}
	R(new(DownloadOptions), "k8s-container-registry-download-image", "Download container image to a file", func(s *mcclient.ClientSession, args *DownloadOptions) error {
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
			name := parts[0]
			tag := parts[1]
			fileName, src, size, err = k8s.ContainerRegistries.DownloadImage(s, args.Registry, name, tag)
			if err != nil {
				return errors.Wrap(err, "download chart")
			}
		} else {
			fileName, src, size, err = k8s.ContainerRegistries.DownloadImageByManager(s, &k8s.DownloadImageByManagerInput{
				Insecure: args.Insecure,
				Image:    args.NAME,
				Username: args.Username,
				Password: args.Password,
			})
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
		var sink io.Writer = f
		bar := pb.Full.Start64(size)
		barReader := bar.NewProxyReader(src)
		if _, err := io.Copy(sink, barReader); err != nil {
			return errors.Wrap(err, "save chart")
		}
		return nil
	})
}
