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
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/yunionagent"
)

func init() {
	type CopyrightUpdateOptions struct {
		Copyright string  `help:"The copyright"`
		Email     string  `help:"The Email"`
		Docs      *string `help:"the Docs website address"`
		License   *string `help:"the license generator website address"`
		BrandCn   *string `help:"the brand name in Chinese"`
		BrandEn   *string `help:"the brand name in English"`
	}

	R(&CopyrightUpdateOptions{}, "copyright-update", "update copyright", func(s *mcclient.ClientSession, args *CopyrightUpdateOptions) error {
		if !s.HasSystemAdminPrivilege() {
			return fmt.Errorf("require admin privilege")
		}

		params := jsonutils.NewDict()
		if args.Docs != nil {
			params.Add(jsonutils.NewString(*args.Docs), "docs")
		}

		if args.License != nil {
			params.Add(jsonutils.NewString(*args.License), "license")
		}

		if args.BrandCn != nil {
			params.Add(jsonutils.NewString(*args.BrandCn), "brand_cn")
		}

		if args.BrandEn != nil {
			params.Add(jsonutils.NewString(*args.BrandEn), "brand_en")
		}

		if len(args.Copyright) > 0 {
			params.Add(jsonutils.NewString(args.Copyright), "copyright")
		}

		if len(args.Email) > 0 {
			params.Add(jsonutils.NewString(args.Email), "email")
		}

		r, err := yunionagent.Copyright.Update(s, "copyright", params)
		if err != nil {
			return err
		}

		printObject(r)
		return nil
	})

	type EnterpriseUpdateOptions struct {
		Name      string `help:"Enterprise name"`
		Copyright string `help:"The Email"`
		Logo      string `help:"logo file path"`
		LoginLogo string `help:"login page logo file path"`
		Favicon   string `help:"favicon path"`
	}

	R(&EnterpriseUpdateOptions{}, "infos-update", "update enterprise info", func(s *mcclient.ClientSession, args *EnterpriseUpdateOptions) error {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		if len(args.Name) > 0 {
			w.WriteField("name", args.Name)
		}

		if len(args.Copyright) > 0 {
			w.WriteField("copyright", args.Copyright)
		}

		writeFile := func(filedName, filePath string) error {
			if len(filePath) == 0 {
				return nil
			}

			c, err := ioutil.ReadFile(filePath)
			if err != nil {
				return err
			}

			var v string
			c64 := base64.StdEncoding.EncodeToString(c)
			ext := filepath.Ext(filePath)
			switch ext {
			case ".png":
				v = fmt.Sprintf("data:image/png;base64,%s", c64)
			case ".jpg":
				v = fmt.Sprintf("data:image/jpg;base64,%s", c64)
			default:
				return fmt.Errorf("only support png/jpg image")
			}

			err = w.WriteField(filedName, v)
			if err != nil {
				return err
			}

			return nil
		}

		if err := writeFile("logo", args.Logo); err != nil {
			return err
		}

		if err := writeFile("login_logo", args.LoginLogo); err != nil {
			return err
		}

		if err := writeFile("favicon", args.Favicon); err != nil {
			return err
		}

		header := make(http.Header)
		header.Set("content-type", fmt.Sprintf("multipart/form-data; boundary=%s", w.Boundary()))
		w.Close()

		ret, err := yunionagent.Info.Update(s, header, bufio.NewReader(&b))
		if err != nil {
			return err
		}

		printObject(ret)
		return nil
	})
}
