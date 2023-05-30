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

package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initRelease() {
	cmd := NewK8sResourceCmd(k8s.Releases)
	cmd.List(new(o.ReleaseListOptions))
	cmd.Create(new(o.ReleaseCreateOptions))
	cmd.Delete(new(o.ReleaseDeleteOptions))
	cmd.ShowEvent()

	cmdN := func(suffix string) string {
		return resourceCmdN("release", suffix)
	}
	R(&o.NamespaceResourceGetOptions{}, cmdN("show"), "Get helm release details", func(s *mcclient.ClientSession, args *o.NamespaceResourceGetOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.Releases.Get(s, args.NAME, params)
		if err != nil {
			return err
		}
		resources, err := ret.Get("resources")
		if err != nil {
			return err
		}
		printObject(resources)
		return nil
	})

	R(&o.ReleaseUpgradeOptions{}, cmdN("upgrade"), "Upgrade release", func(s *mcclient.ClientSession, args *o.ReleaseUpgradeOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}

		res, err := k8s.Releases.PerformAction(s, args.NAME, "upgrade", params)
		if err != nil {
			return err
		}
		printObject(res)
		return nil
	})

	R(&o.ReleaseHistoryOptions{}, cmdN("history"), "Get release history", func(s *mcclient.ClientSession, args *o.ReleaseHistoryOptions) error {
		ret, err := k8s.Releases.GetSpecific(s, args.NAME, "history", args.Params())
		if err != nil {
			return err
		}
		printObjectYAML(ret)
		return nil
	})

	R(&o.ReleaseRollbackOptions{}, cmdN("rollback"), "Rollback release by history revision number", func(s *mcclient.ClientSession, args *o.ReleaseRollbackOptions) error {
		ret, err := k8s.Releases.PerformAction(s, args.NAME, "rollback", args.Params())
		if err != nil {
			return err
		}
		printObjectYAML(ret)
		return nil
	})
}
