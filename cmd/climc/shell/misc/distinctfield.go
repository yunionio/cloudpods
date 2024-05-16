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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

func init() {
	type DistinctFieldOption struct {
		MODULE string `help:"module name"`
		FIELD  string `help:"distinct field name to query"`
		Extra  bool   `help:"query extra distinct field"`
	}
	R(&DistinctFieldOption{}, "distinct-field", "Query values of a distinct field for a module", func(s *mcclient.ClientSession, args *DistinctFieldOption) error {
		mod, err := modulebase.GetModule(s, args.MODULE)
		if err != nil || mod == nil {
			if err != nil {
				return fmt.Errorf("module %s not found %s", args.MODULE, err)
			}
			return fmt.Errorf("No module %s found", args.MODULE)
		}
		params := jsonutils.NewDict()
		if args.Extra {
			params.Add(jsonutils.NewString(args.FIELD), "extra_field")
		} else {
			params.Add(jsonutils.NewString(args.FIELD), "field")
		}
		result, err := mod.Get(s, "distinct-field", params)
		if err != nil {
			return err
		}
		fmt.Println(result)
		return nil
	})

	type DistinctFieldsOption struct {
		MODULE        string   `help:"module name"`
		Field         []string `help:"distinct field name to query"`
		ExtraField    []string `help:"distinct field name to query"`
		ExtraResource string
	}
	R(&DistinctFieldsOption{}, "distinct-fields", "Query values of a distinct fields for a module", func(s *mcclient.ClientSession, args *DistinctFieldsOption) error {
		mod, err := modulebase.GetModule(s, args.MODULE)
		if err != nil || mod == nil {
			if err != nil {
				return fmt.Errorf("module %s not found %s", args.MODULE, err)
			}
			return fmt.Errorf("No module %s found", args.MODULE)
		}
		params := jsonutils.Marshal(args)
		result, err := mod.Get(s, "distinct-fields", params)
		if err != nil {
			return err
		}
		fmt.Println(result)
		return nil
	})

}
