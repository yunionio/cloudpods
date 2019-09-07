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
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	/**
	 * 新建一个标签
	 */
	type LabelCreateOptions struct {
		NAME      string `help:"Name of the new label"`
		VALUETYPE string `help:"Label value's type such as int, float, string, bool, choices"`
		Remark    string `help:"Remark or description of the new label"`
	}
	R(&LabelCreateOptions{}, "label-create", "Create a label", func(s *mcclient.ClientSession, args *LabelCreateOptions) error {
		mod, err := modulebase.GetModule(s, "labels")
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.VALUETYPE), "value_type")
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}

		label, err := mod.Create(s, params)
		if err != nil {
			return err
		}
		printObject(label)
		return nil
	})

	/**
	 * 删除指定name的标签（删除指定name的标签）
	 */
	type LabelDeleteOptions struct {
		NAME string `help:"Name of label"`
	}
	R(&LabelDeleteOptions{}, "label-delete", "Delete a label", func(s *mcclient.ClientSession, args *LabelDeleteOptions) error {
		label, e := modules.Labels.Delete(s, args.NAME, nil)
		if e != nil {
			return e
		}
		printObject(label)
		return nil
	})

	/**
	 * 修改标签（修改指定name的标签）
	 */
	type LabelUpdateOptions struct {
		NAME      string `help:"Name of the label"`
		ValueType string `help:"Label value's type such as int, float, string, bool, choices"`
		Remark    string `help:"Remark or description of the new label"`
	}
	R(&LabelUpdateOptions{}, "label-update", "Update a label", func(s *mcclient.ClientSession, args *LabelUpdateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		if len(args.ValueType) > 0 {
			params.Add(jsonutils.NewString(args.ValueType), "value_type")
		}
		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}

		label, err := modules.Labels.Patch(s, args.NAME, params)
		if err != nil {
			return err
		}
		printObject(label)
		return nil
	})

	/**
	 * 列出所有的标签
	 */
	type LabelListOptions struct {
		options.BaseListOptions
	}
	R(&LabelListOptions{}, "label-list", "List labels", func(s *mcclient.ClientSession, args *LabelListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Labels.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Labels.GetColumns(s))
		return nil
	})

}
