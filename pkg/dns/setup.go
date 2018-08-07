package dns

import (
	"fmt"
	"os"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/mholt/caddy"

	"github.com/yunionio/pkg/util/regutils"
)

func init() {
	caddy.RegisterPlugin(PluginName, caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	os.Stderr = os.Stdout

	rDNS, err := regionDNSParse(c)
	if err != nil {
		return plugin.Error(PluginName, err)
	}

	if rDNS.PrimaryZone == "" {
		return fmt.Errorf("dns_domain must provided")
	}
	if !regutils.MatchDomainName(rDNS.PrimaryZone) {
		return fmt.Errorf("dns_domain %q not match domain format", rDNS.PrimaryZone)
	}

	err = rDNS.initDB(c)
	if err != nil {
		return plugin.Error(PluginName, err)
	}

	rDNS.initK8s(c)

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
				case "kube_config":
					if !c.NextArg() {
						return nil, c.ArgErr()
					}
					rDNS.K8sConfigFile = c.Val()
				case "upstream":
					args := c.RemainingArgs()
					u, err := upstream.New(args)
					if err != nil {
						return nil, err
					}
					rDNS.Upstream = u
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
