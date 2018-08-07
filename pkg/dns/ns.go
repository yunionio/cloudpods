package dns

import (
	//"net"
	"strings"

	"github.com/miekg/dns"
)

const defaultNSName = "ns.dns."

func isDefaultNS(name, zone string) bool {
	return strings.Index(name, defaultNSName) == 0 && strings.Index(name, zone) == len(defaultNSName)
}

func (r *SRegionDNS) nsAddr() *dns.A {
	//rr := new(dns.A)
	return nil
}
