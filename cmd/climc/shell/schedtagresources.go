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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type schedtagModelHelper struct {
	managers []modulebase.JointResourceManager
}

func newSchedtagModelHelper(mans ...modulebase.JointResourceManager) *schedtagModelHelper {
	return &schedtagModelHelper{managers: mans}
}

func (h *schedtagModelHelper) register() {
	for _, man := range h.managers {
		h.list(man.Slave, man.Slave.GetKeyword())
		h.add(man, man.Slave.GetKeyword())
		h.remove(man, man.Slave.GetKeyword())
		h.setTags(man, man.Slave.GetKeyword())
	}
}

func (h *schedtagModelHelper) list(slave modulebase.Manager, kw string) {
	R(
		&options.SchedtagModelListOptions{},
		fmt.Sprintf("schedtag-%s-list", kw),
		fmt.Sprintf("List all scheduler tag and %s pairs", kw),
		func(s *mcclient.ClientSession, args *options.SchedtagModelListOptions) error {
			mod, err := modulebase.GetJointModule2(s, &modules.Schedtags, slave)
			if err != nil {
				return err
			}
			params, err := args.Params()
			if err != nil {
				return err
			}
			var result *modulebase.ListResult
			if len(args.Schedtag) > 0 {
				result, err = mod.ListDescendent(s, args.Schedtag, params)
			} else {
				result, err = mod.List(s, params)
			}
			if err != nil {
				return err
			}
			printList(result, mod.GetColumns(s))
			return nil
		},
	)
}

func (h *schedtagModelHelper) add(man modulebase.JointResourceManager, kw string) {
	R(
		&options.SchedtagModelPairOptions{},
		fmt.Sprintf("schedtag-%s-add", kw),
		fmt.Sprintf("Add a schedtag to a %s", kw),
		func(s *mcclient.ClientSession, args *options.SchedtagModelPairOptions) error {
			schedtag, err := man.Attach(s, args.SCHEDTAG, args.OBJECT, nil)
			if err != nil {
				return err
			}
			printObject(schedtag)
			return nil
		})
}

func (h *schedtagModelHelper) remove(man modulebase.JointResourceManager, kw string) {
	R(
		&options.SchedtagModelPairOptions{},
		fmt.Sprintf("schedtag-%s-remove", kw),
		fmt.Sprintf("Remove a schedtag to a %s", kw),
		func(s *mcclient.ClientSession, args *options.SchedtagModelPairOptions) error {
			schedtag, err := man.Detach(s, args.SCHEDTAG, args.OBJECT, nil)
			if err != nil {
				return err
			}
			printObject(schedtag)
			return nil
		})
}

func (h *schedtagModelHelper) setTags(man modulebase.JointResourceManager, kw string) {
	R(
		&options.SchedtagSetOptions{},
		fmt.Sprintf("%s-set-schedtag", kw),
		fmt.Sprintf("Set schedtags to %v", kw),
		func(s *mcclient.ClientSession, args *options.SchedtagSetOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := man.Slave.PerformAction(s, args.ID, "set-schedtag", params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
}

func init() {
	newSchedtagModelHelper(
		modules.Schedtaghosts,
		modules.Schedtagstorages,
		modules.Schedtagnetworks,
	).register()
}
