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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type CopyrightUpdateOptions struct {
		Copyright string  `help:"The copyright"`
		Email     string  `help:"The Email"`
		Docs      *string `help:"the Docs website address"`
	}

	R(&CopyrightUpdateOptions{}, "copyright-update", "update copyright", func(s *mcclient.ClientSession, args *CopyrightUpdateOptions) error {
		if !s.HasSystemAdminPrivilege() {
			return fmt.Errorf("require admin privilege")
		}

		params := jsonutils.NewDict()
		if args.Docs != nil {
			params.Add(jsonutils.NewString(*args.Docs), "docs")
		}

		if len(args.Copyright) > 0 {
			params.Add(jsonutils.NewString(args.Copyright), "copyright")
		}

		if len(args.Email) > 0 {
			params.Add(jsonutils.NewString(args.Email), "email")
		}

		r, err := modules.Copyright.Update(s, "copyright", params)
		if err != nil {
			return err
		}

		printObject(r)
		return nil
	})
}
