package dns

import (
	"os"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/mholt/caddy"

	"github.com/yunionio/log"
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
	log.Infof("regionDNSParse succ: %#v", rDNS)

	err = rDNS.initDB(c)
	if err != nil {
		return plugin.Error(PluginName, err)
	}

	err = rDNS.initK8s(c)
	if err != nil {
		return plugin.Error(PluginName, err)
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
		log.Warningf("==zones: %v", rDNS.Zones)

		if c.NextBlock() {
			for {
				log.Printf("===val: %v", c.Val())
				switch c.Val() {
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
