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

package misc

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type LicenseListOptions struct {
		options.BaseListOptions
	}

	R(&LicenseListOptions{}, "licenses-list", "show licenses", func(s *mcclient.ClientSession, args *LicenseListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}

		lics, err := modules.License.List(s, params)
		if err != nil {
			return err
		}

		printList(lics, modules.License.GetColumns(s))
		return nil
	})

	type LicenseUploadOptions struct {
		FILE string
	}

	R(&LicenseUploadOptions{}, "license-upload", "Upload license", func(s *mcclient.ClientSession, args *LicenseUploadOptions) error {
		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		fileWriter, err := bodyWriter.CreateFormFile("license", args.FILE)
		if err != nil {
			return err
		}
		fh, err := os.Open(args.FILE)
		if err != nil {
			return err
		}
		defer fh.Close()
		_, err = io.Copy(fileWriter, fh)
		if err != nil {
			return err
		}
		contentType := bodyWriter.FormDataContentType()
		bodyWriter.Close()

		header := http.Header{}
		header.Set("Content-Type", contentType)
		lic, err := modules.License.Upload(s, header, bodyBuf)
		if err != nil {
			return err
		}

		printObject(lic)
		return nil
	})

	type LicenseShowOptions struct {
		SERVICE string `help:"service name"  choices:"compute|service_tree"`
	}

	R(&LicenseShowOptions{}, "licenses-show", "show actived license", func(s *mcclient.ClientSession, args *LicenseShowOptions) error {
		lic, e := modules.License.Get(s, args.SERVICE, nil)
		if e != nil {
			return e
		}

		printObject(lic)
		return nil
	})

	type LicenseStatusOptions struct {
		SERVICE string `help:"service name"  choices:"compute|service_tree"`
	}

	R(&LicenseStatusOptions{}, "licenses-usage", "show license usages status", func(s *mcclient.ClientSession, args *LicenseStatusOptions) error {
		status, err := modules.License.GetSpecific(s, args.SERVICE, "status", nil)
		if err != nil {
			return err
		}

		printObject(status)
		return nil
	})

}
