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

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/miekg/dns"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

var log = clog.NewWithPlugin(PluginName)

func init() {
	plugin.Register(PluginName, setup)
}

func setup(c *caddy.Controller) error {
	for c.Next() {
		log.Infof("setup: %s", PluginName)

		var fallF fall.F
		upStream := upstream.New()

		rDNS := New()
		rDNS.Zones = c.RemainingArgs()
		if len(rDNS.Zones) == 0 {
			rDNS.Zones = make([]string, len(c.ServerBlockKeys))
			copy(rDNS.Zones, c.ServerBlockKeys)
		}
		for i, str := range rDNS.Zones {
			rDNS.Zones[i] = plugin.Host(str).Normalize()
		}

		for c.NextBlock() {
			switch c.Val() {
			case "dns_domain":
				if !c.NextArg() {
					return plugin.Error(PluginName+".dns_domain", c.ArgErr())
				}
				rDNS.PrimaryZone = c.Val()
			case "sql_connection":
				if !c.NextArg() {
					return plugin.Error(PluginName+".sql_connection", c.ArgErr())
				}
				rDNS.SqlConnection = c.Val()
			case "auth_url":
				if !c.NextArg() {
					return plugin.Error(PluginName+".auth_url", c.ArgErr())
				}
				rDNS.AuthUrl = c.Val()
			case "admin_project":
				if !c.NextArg() {
					return plugin.Error(PluginName+".admin_project", c.ArgErr())
				}
				rDNS.AdminProject = c.Val()
			case "admin_project_domain":
				if !c.NextArg() {
					return plugin.Error(PluginName+".admin_project_domain", c.ArgErr())
				}
				rDNS.AdminProjectDomain = c.Val()
			case "admin_user":
				if !c.NextArg() {
					return plugin.Error(PluginName+".admin_user", c.ArgErr())
				}
				rDNS.AdminUser = c.Val()
			case "admin_domain":
				if !c.NextArg() {
					return plugin.Error(PluginName+".admin_domain", c.ArgErr())
				}
				rDNS.AdminDomain = c.Val()
			case "admin_password":
				if !c.NextArg() {
					return plugin.Error(PluginName+".admin_password", c.ArgErr())
				}
				rDNS.AdminPassword = c.Val()
			case "region":
				if !c.NextArg() {
					return plugin.Error(PluginName+".region", c.ArgErr())
				}
				rDNS.Region = c.Val()
			case "in_cloud_only":
				rDNS.InCloudOnly = true
			case "upstream":
				c.RemainingArgs()
			case "fallthrough":
				fallF.SetZonesFromArgs(c.RemainingArgs())
			default:
				return plugin.Error(PluginName, c.Errf("unknown property %q", c.Val()))
			}
		}

		ctx, cancel := context.WithCancel(context.Background())

		rDNS.initAuth()

		conf, err := identity.ServicesV3.GetConfig(rDNS.getAdminSession(ctx), apis.SERVICE_TYPE_REGION)
		if err != nil {
			cancel()
			return errors.Wrap(err, "GetConfig")
		}

		if conf.Contains("dns_domain") {
			rDNS.PrimaryZone, _ = conf.GetString("dns_domain")
		}
		log.Infof("use dns_domain %q", rDNS.PrimaryZone)

		if len(rDNS.PrimaryZone) > 0 && !regutils.MatchDomainName(rDNS.PrimaryZone) {
			cancel()
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
			cancel()
			return plugin.Error(PluginName, err)
		}

		rDNS.Fall = fallF
		rDNS.Upstream = upStream

		dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
			rDNS.Next = next
			return rDNS
		})
		c.OnShutdown(func() error { cancel(); return nil })
	}

	return nil
}
