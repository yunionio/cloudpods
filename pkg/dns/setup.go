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
	"context"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
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

	rDNS.initAuth()

	conf, err := identity.ServicesV3.GetConfig(rDNS.getAdminSession(context.Background()), apis.SERVICE_TYPE_REGION)
	if err != nil {
		return errors.Wrap(err, "GetConfig")
	}

	if conf.Contains("dns_domain") {
		rDNS.PrimaryZone, _ = conf.GetString("dns_domain")
	}
	log.Infof("use dns_domain %q", rDNS.PrimaryZone)

	if len(rDNS.PrimaryZone) > 0 && !regutils.MatchDomainName(rDNS.PrimaryZone) {
		return errors.Wrapf(errors.ErrInvalidFormat, "dns_domain %q invalid", rDNS.PrimaryZone)
	}
	if len(rDNS.PrimaryZone) > 0 {
		if r := rDNS.PrimaryZone[len(rDNS.PrimaryZone)-1]; r != '.' {
			rDNS.PrimaryZone += "."
		}
	} else {
		rDNS.PrimaryZone = ""
	}
	rDNS.primaryZoneLabelCount = dns.CountLabel(rDNS.PrimaryZone)

	err = rDNS.initDB(c)
	if err != nil {
		return plugin.Error(PluginName, err)
	}

	/*if !rDNS.K8sSkip {
		go rDNS.initK8s()
	}*/

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
				case "admin_project_domain":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.AdminProjectDomain = c.Val()
				case "admin_user":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.AdminUser = c.Val()
				case "admin_domain":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.AdminDomain = c.Val()
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
				case "in_cloud_only":
					rDNS.InCloudOnly = true
				// case "k8s_skip":
				//	rDNS.K8sSkip = true
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
