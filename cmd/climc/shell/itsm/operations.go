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

package itsm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/itsm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type OperationListOptions struct {
		options.BaseListOptions
	}
	R(&OperationListOptions{}, "operation-list", "List operations", func(s *mcclient.ClientSession, suboptions *OperationListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = suboptions.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		result, err := modules.Operations.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Operations.GetColumns(s))
		return nil
	})

	type OperationUpdateOptions struct {
		TASK_ID string `help:"TASK_ID of process" `
		Content string `help:"Content of process" default:""`
		Result  string `help:"Result of process" default:""`
	}
	R(&OperationUpdateOptions{}, "operation-update", "Update operation", func(s *mcclient.ClientSession, args *OperationUpdateOptions) error {
		params := jsonutils.NewDict()

		// params.Add(jsonutils.NewString(args.TASK_ID), "task_id")
		params.Add(jsonutils.NewString(args.Content), "content")
		params.Add(jsonutils.NewString(args.Result), "result")

		obj, err := modules.Operations.Put(s, args.TASK_ID, params)
		if err != nil {
			return err
		}
		printObject(obj)

		return nil
	})

	type OperationShowOptions struct {
		INSTANCE_ID string `help:"INSTANCE_ID of process" `
	}

	R(&OperationShowOptions{}, "operation-show", "Show operation details", func(s *mcclient.ClientSession, args *OperationShowOptions) error {
		result, err := modules.Operations.Get(s, args.INSTANCE_ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type OperationCreateOptions struct {
		DeviceCode   string `help:"DeviceCode of process"`
		ResourceSum  int64  `help:"ResourceSum of process, default: 0" default:"0"`
		TemplateId   string `help:"TemplateId of process" `
		Cpu          int64  `help:"Cpu of process, default: 0" default:"0"`
		Memery       int64  `help:"Memery of process, default: 0" default:"0"`
		Disk         int64  `help:"Disk of process, default: 0" default:"0"`
		ProjectId    string `help:"ProjectId of process" `
		Remark       string `help:"Remark of process" `
		OPERATE_TYPE string `help:"OPERATE_TYPE of process" choices:"apply|config|template|delete"`
		Schedule     string `help:"Schedule of process"`
	}

	R(&OperationCreateOptions{}, "operation-create", "Create a operation", func(s *mcclient.ClientSession, args *OperationCreateOptions) error {
		params := jsonutils.NewDict()

		params.Add(jsonutils.NewString(args.OPERATE_TYPE), "operate_type")

		if len(args.DeviceCode) > 0 {
			params.Add(jsonutils.NewString(args.DeviceCode), "device_code")
		}

		if args.ResourceSum > 0 {
			params.Add(jsonutils.NewInt(args.ResourceSum), "resource_sum")
		}

		if len(args.TemplateId) > 0 {
			params.Add(jsonutils.NewString(args.TemplateId), "template_id")
		}

		if args.Cpu > 0 {
			params.Add(jsonutils.NewInt(args.Cpu), "cpu")
		}

		if args.Memery > 0 {
			params.Add(jsonutils.NewInt(args.Memery), "memery")
		}

		if args.Disk > 0 {
			params.Add(jsonutils.NewInt(args.Disk), "disk")
		}

		if len(args.ProjectId) > 0 {
			params.Add(jsonutils.NewString(args.ProjectId), "project_id")
		}

		if len(args.Remark) > 0 {
			params.Add(jsonutils.NewString(args.Remark), "remark")
		}

		if len(args.Schedule) > 0 {
			params.Add(jsonutils.NewString(args.Schedule), "schedule")
		}

		operation, err := modules.Operations.Create(s, params)
		if err != nil {
			return err
		}
		printObject(operation)
		return nil
	})

}
