package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
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
		mod, err := modules.GetModule(s, "labels")
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
		BaseListOptions
	}
	R(&LabelListOptions{}, "label-list", "List labels", func(s *mcclient.ClientSession, args *LabelListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		result, err := modules.Labels.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Labels.GetColumns(s))
		return nil
	})

}
