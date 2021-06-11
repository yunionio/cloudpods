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

package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/lbagent/gobetween"
	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
)

type GenGobetweenConfigOptions struct {
	LoadbalancersEnabled []*Loadbalancer
	AgentParams          *AgentParams

	Config *gobetween.Config
}

func (b *LoadbalancerCorpus) GenGobetweenConfigs(dir string, opts *GenGobetweenConfigOptions) error {
	//agentParams := opts.AgentParams
	// TODO
	//  - dynamic password
	//  - log to remote
	//  - log to local syslog unix sock
	//  - respawn
	opts.Config = &gobetween.Config{
		Servers: map[string]gobetween.Server{},
		Api: gobetween.ApiConfig{
			Enabled: true,
			Bind:    "localhost:777",
			BasicAuth: &gobetween.ApiBasicAuthConfig{
				Login:    "Yunion",
				Password: "LBStats",
			},
		},
	}
	for _, lb := range opts.LoadbalancersEnabled {
		for _, listener := range lb.Listeners {
			if listener.ListenerType != "udp" {
				continue
			}
			if listener.Status != "enabled" {
				continue
			}
			if listener.BackendGroupId == "" {
				continue
			}
			backendGroup := lb.BackendGroups[listener.BackendGroupId]
			if backendGroup == nil || len(backendGroup.Backends) == 0 {
				continue
			}

			// backends
			staticList := []string{}
			for _, backend := range backendGroup.Backends {
				backendS := fmt.Sprintf("%s:%d weight=%d", backend.Address, backend.Port, backend.Weight)
				staticList = append(staticList, backendS)
			}

			// scheduler
			var serverBalance string
			switch listener.Scheduler {
			case "rr", "wrr":
				serverBalance = "roundrobin"
			case "lc", "wlc":
				serverBalance = "leastconn"
			case "sch", "tch":
				log.Warningf("scheduler %s converted to iphash1", listener.Scheduler)
				serverBalance = "iphash1"
			default:
				log.Warningf("scheduler %s converted to iphash1", listener.Scheduler)
				serverBalance = "iphash1"
			}

			// healthcheck
			var serverHealthcheck *gobetween.HealthcheckConfig
			if listener.HealthCheck == "on" && listener.HealthCheckType == "udp" {
				serverHealthcheck = &gobetween.HealthcheckConfig{
					Kind:     "pingudp",
					Interval: fmt.Sprintf("%ds", listener.HealthCheckInterval),
					Timeout:  fmt.Sprintf("%ds", listener.HealthCheckTimeout),
					Passes:   listener.HealthCheckRise,
					Fails:    listener.HealthCheckFall,
					UdpHealthcheckConfig: &gobetween.UdpHealthcheckConfig{
						Send:    listener.HealthCheckReq,
						Receive: listener.HealthCheckExp,
					},
				}
			}

			// acl
			var serverAccess *gobetween.AccessConfig
			if listener.AclStatus == "on" && listener.AclId != "" {
				acl := b.LoadbalancerAcls[listener.AclId]
				if acl == nil {
					log.Warningf("listener %s(%s): unknown acl %s",
						listener.Name, listener.Id, listener.AclId)
					continue
				}
				var accessDefault string
				var accessRulesAction string
				accessRules := []string{}
				switch listener.AclType {
				case "black":
					accessDefault = "allow"
					accessRulesAction = "deny"
				case "white":
					accessDefault = "deny"
					accessRulesAction = "allow"
				default:
					log.Warningf("listener %s(%s): unknown acl type: %s",
						listener.Name, listener.Id, listener.AclId)
					continue
				}
				for _, aclEntry := range *acl.AclEntries {
					rule := fmt.Sprintf("%s %s", accessRulesAction, aclEntry.Cidr)
					accessRules = append(accessRules, rule)
				}
				serverAccess = &gobetween.AccessConfig{
					Default: accessDefault,
					Rules:   accessRules,
				}
			}
			pize := func(s string) *string {
				return &s
			}

			opts.Config.Servers[listener.Id] = gobetween.Server{
				Bind:     fmt.Sprintf("%s:%d", lb.Address, listener.ListenerPort),
				Protocol: "udp",
				Balance:  serverBalance,
				Discovery: &gobetween.DiscoveryConfig{
					Kind: "static",
					StaticDiscoveryConfig: &gobetween.StaticDiscoveryConfig{
						StaticList: staticList,
					},
				},
				Healthcheck: serverHealthcheck,
				Access:      serverAccess,
				ConnectionOptions: gobetween.ConnectionOptions{
					ClientIdleTimeout:        pize(fmt.Sprintf("%ds", listener.ClientIdleTimeout)),
					BackendIdleTimeout:       pize(fmt.Sprintf("%ds", listener.BackendIdleTimeout)),
					BackendConnectionTimeout: pize(fmt.Sprintf("%ds", listener.BackendConnectTimeout)),
				},
			}
		}
	}
	{
		// write gobetween.json
		d, err := json.MarshalIndent(opts.Config, "", "  ")
		if err != nil {
			return err
		}
		p := filepath.Join(dir, "gobetween.json")
		err = ioutil.WriteFile(p, d, agentutils.FileModeFile)
		if err != nil {
			return err
		}
	}
	return nil
}
