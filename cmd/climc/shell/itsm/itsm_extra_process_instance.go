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
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type ExtraProcessInstanceListOptions struct {
		ID     string `help:"ID of ExtraProcessInstance"`
		UserId string `help:"ID of user"`
	}
	R(&ExtraProcessInstanceListOptions{}, "itsm_extra_process_instance_list", "extra process instance list",
		func(s *mcclient.ClientSession, args *ExtraProcessInstanceListOptions) error {
			params := jsonutils.NewDict()
			if len(args.ID) > 0 {
				params.Add(jsonutils.NewString(args.ID), "id")
			}
			if len(args.UserId) > 0 {
				params.Add(jsonutils.NewString(args.UserId), "user_id")
			}

			ret, err := modules.ExtraProcessInstance.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, modules.ExtraProcessInstance.GetColumns(s))
			return nil
		})

	type ExtraProcessInstanceShowOptions struct {
		ID string `help:"ID of ExtraProcessInstance" required:"true"`
	}
	R(&ExtraProcessInstanceShowOptions{}, "itsm_extra_process_instance_list", "extra process instance list",
		func(s *mcclient.ClientSession, args *ExtraProcessInstanceShowOptions) error {
			ret, err := modules.ExtraProcessInstance.Get(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
}
