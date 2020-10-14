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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

func init() {
	type CurrentUserOptions struct {
	}
	R(&CurrentUserOptions{}, "session-show", "show information of current account", func(s *mcclient.ClientSession, args *CurrentUserOptions) error {
		printObject(s.ToJson())
		return nil
	})
	type FreezeResrouceOptions struct {
		ID     string `help:"ID of resource"`
		Module string `help:"Resource module type, eg: servers, disks, networks"`
	}
	R(&FreezeResrouceOptions{}, "freeze-resource", "Freeze resource operation update and perform action except for unfreeze", func(s *mcclient.ClientSession, args *FreezeResrouceOptions) error {
		mod, err := modulebase.GetModule(s, args.Module)
		if err != nil {
			return errors.Wrap(err, "failed get module")
		}
		obj, err := mod.PerformAction(s, args.ID, "freeze", nil)
		if err != nil {
			return err
		}
		printObject(obj)
		return nil
	})
	type UnfreezeResrouceOptions struct {
		ID     string `help:"ID of resource"`
		Module string `help:"Resource module type, eg: servers, disks, networks"`
	}
	R(&UnfreezeResrouceOptions{}, "unfreeze-resource", "Unfreeze resource", func(s *mcclient.ClientSession, args *UnfreezeResrouceOptions) error {
		mod, err := modulebase.GetModule(s, args.Module)
		if err != nil {
			return errors.Wrap(err, "failed get module")
		}
		obj, err := mod.PerformAction(s, args.ID, "unfreeze", nil)
		if err != nil {
			return err
		}
		printObject(obj)
		return nil
	})
}
