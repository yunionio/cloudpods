package main

import (
	_ "github.com/coredns/coredns/plugin/cache"
	_ "github.com/coredns/coredns/plugin/chaos"
	_ "github.com/coredns/coredns/plugin/debug"
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/file"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/health"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/metrics"
	_ "github.com/coredns/coredns/plugin/nsid"
	_ "github.com/coredns/coredns/plugin/proxy"
	_ "github.com/coredns/coredns/plugin/reload"
	_ "github.com/coredns/coredns/plugin/trace"
	_ "github.com/coredns/coredns/plugin/whoami"
	_ "github.com/mholt/caddy/startupshutdown"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"

	_ "github.com/yunionio/onecloud/pkg/dns"
)

var directives = []string{
	"debug",
	"trace",
	"health",
	"errors",
	"log",
	"chaos",
	"cache",
	"hosts",
	"file",
	"yunion",
	"forward",
	"proxy",
	"whoami",
	"startup",
	"shutdown",
}

func init() {
	dnsserver.Directives = directives
}

func main() {
	coremain.Run()
}
