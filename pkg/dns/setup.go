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

package dns

import (
	"fmt"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/mholt/caddy"

	"yunion.io/x/pkg/util/regutils"
)

func init() {
	caddy.RegisterPlugin(PluginName, caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	rDNS, err := regionDNSParse(c)
	if err != nil {
		return plugin.Error(PluginName, err)
	}

	if len(rDNS.PrimaryZone) == 0 {
		return fmt.Errorf("dns_domain missing")
	}
	if !regutils.MatchDomainName(rDNS.PrimaryZone) {
		return fmt.Errorf("dns_domain %q invalid", rDNS.PrimaryZone)
	}

	err = rDNS.initDB(c)
	if err != nil {
		return plugin.Error(PluginName, err)
	}

	if !rDNS.K8sSkip {
		go rDNS.initK8s()
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		rDNS.Next = next
		return rDNS
	})

	return nil
}

func regionDNSParse(c *caddy.Controller) (*SRegionDNS, error) {
	return parseConfig(c)
}

func parseConfig(c *caddy.Controller) (*SRegionDNS, error) {
	rDNS := New()
	for c.Next() {
		rDNS.Zones = c.RemainingArgs()
		if len(rDNS.Zones) == 0 {
			rDNS.Zones = make([]string, len(c.ServerBlockKeys))
			copy(rDNS.Zones, c.ServerBlockKeys)
		}
		for i, str := range rDNS.Zones {
			rDNS.Zones[i] = plugin.Host(str).Normalize()
		}

		if c.NextBlock() {
			for {
				switch c.Val() {
				case "dns_domain":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.PrimaryZone = c.Val()
				case "fallthrough":
					rDNS.Fall.SetZonesFromArgs(c.RemainingArgs())
				case "sql_connection":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.SqlConnection = c.Val()
				case "auth_url":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.AuthUrl = c.Val()
				case "admin_project":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.AdminProject = c.Val()
				case "admin_user":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.AdminUser = c.Val()
				case "admin_password":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.AdminPassword = c.Val()
				case "region":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.Region = c.Val()
				case "upstream":
					args := c.RemainingArgs()
					u, err := upstream.New(args)
					if err != nil {
						return nil, err
					}
					rDNS.Upstream = u
				case "k8s_skip":
					rDNS.K8sSkip = true
				default:
					if c.Val() != "}" {
						return nil, c.Errf("unknown property %q", c.Val())
					}
				}

				if !c.Next() {
					break
				}
			}
		}
	}
	return rDNS, nil
}
