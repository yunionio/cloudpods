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
	"encoding/base64"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	printLbagent := func(data jsonutils.JSONObject) {
		printObjectRecursiveEx(data, func(data jsonutils.JSONObject) {
			// "base64 -d" config template
			keys := []string{
				"params.keepalived_conf_tmpl",
				"params.haproxy_conf_tmpl",
				"params.telegraf_conf_tmpl",
			}
			d := data.(*jsonutils.JSONDict)
			for _, key := range keys {
				if d.Contains(key) {
					s0, _ := d.GetString(key)
					b1, err := base64.StdEncoding.DecodeString(s0)
					if err != nil {
						log.Errorf("%s: invalid base64 string: %s\n  %s", key, err, s0)
						return
					}
					s1 := string(b1)
					d.Set(key, jsonutils.NewString(s1))
				}
			}
			printObject(d)
		})
	}
	R(&options.LoadbalancerAgentCreateOptions{}, "lbagent-create", "Create lbagent", func(s *mcclient.ClientSession, opts *options.LoadbalancerAgentCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbagent, err := modules.LoadbalancerAgents.Create(s, params)
		if err != nil {
			return err
		}
		printLbagent(lbagent)
		return nil
	})
	R(&options.LoadbalancerAgentGetOptions{}, "lbagent-show", "Show lbagent", func(s *mcclient.ClientSession, opts *options.LoadbalancerAgentGetOptions) error {
		lbagent, err := modules.LoadbalancerAgents.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printLbagent(lbagent)
		return nil
	})
	R(&options.LoadbalancerAgentListOptions{}, "lbagent-list", "List lbagents", func(s *mcclient.ClientSession, opts *options.LoadbalancerAgentListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerAgents.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerAgents.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerAgentUpdateOptions{}, "lbagent-update", "Update lbagent", func(s *mcclient.ClientSession, opts *options.LoadbalancerAgentUpdateOptions) error {
		params, err := options.StructToParams(opts)
		lbagent, err := modules.LoadbalancerAgents.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printLbagent(lbagent)
		return nil
	})
	R(&options.LoadbalancerAgentDeleteOptions{}, "lbagent-delete", "Show lbagent", func(s *mcclient.ClientSession, opts *options.LoadbalancerAgentDeleteOptions) error {
		lbagent, err := modules.LoadbalancerAgents.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printLbagent(lbagent)
		return nil
	})
	R(&options.LoadbalancerAgentActionHbOptions{}, "lbagent-heartbeat", "Emulate a lbagent heartbeat", func(s *mcclient.ClientSession, opts *options.LoadbalancerAgentActionHbOptions) error {
		lbagent, err := modules.LoadbalancerAgents.PerformAction(s, opts.ID, "hb", nil)
		if err != nil {
			return err
		}
		printLbagent(lbagent)
		return nil
	})
	R(&options.LoadbalancerAgentActionPatchParamsOptions{}, "lbagent-params-patch", "Patch lbagent params", func(s *mcclient.ClientSession, opts *options.LoadbalancerAgentActionPatchParamsOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbagent, err := modules.LoadbalancerAgents.PerformAction(s, opts.ID, "params-patch", params)
		if err != nil {
			return err
		}
		printLbagent(lbagent)
		return nil
	})
	R(&options.LoadbalancerAgentActionDeployOptions{}, "lbagent-deploy", "Deploy lbagent", func(s *mcclient.ClientSession, opts *options.LoadbalancerAgentActionDeployOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbagent, err := modules.LoadbalancerAgents.PerformAction(s, opts.ID, "deploy", params)
		if err != nil {
			return err
		}
		printLbagent(lbagent)
		return nil
	})
	R(&options.LoadbalancerAgentActionUndeployOptions{}, "lbagent-undeploy", "Undeploy lbagent", func(s *mcclient.ClientSession, opts *options.LoadbalancerAgentActionUndeployOptions) error {
		lbagent, err := modules.LoadbalancerAgents.PerformAction(s, opts.ID, "undeploy", nil)
		if err != nil {
			return err
		}
		printLbagent(lbagent)
		return nil
	})
}
