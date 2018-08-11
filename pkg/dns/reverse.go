package dns

import (
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/etcd/msg"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/coredns/coredns/request"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/compute/models"
)

// Reverse implements the ServiceBackend interface
func (r *SRegionDNS) Reverse(state request.Request, exact bool, opt plugin.Options) (services []msg.Service, err error) {
	ip := dnsutil.ExtractAddressFromReverse(state.Name())
	if ip == "" {
		_, e := r.Records(state, exact)
		return nil, e
	}
	records, err := r.getNameForIp(ip, state)
	if err != nil {
		log.Errorf("Reverse get name for ip: %v", err)
	}
	if len(records) == 0 {
		return records, errNoItems
	}
	return records, nil
}

func (r *SRegionDNS) getNameForIp(ip string, state request.Request) ([]msg.Service, error) {
	req, e := parseRequest(state)
	if e != nil {
		return nil, e
	}

	// 1. try local dns records table
	records := models.DnsRecordManager.QueryDnsIps(req.ProjectId(), req.Name(), req.Type())
	for _, rec := range records {
		return []msg.Service{{Host: rec.Addr, TTL: uint32(rec.Ttl)}}, nil
	}

	// 2. try hosts table
	host := models.HostnetworkManager.GetHostByAddress(ip)
	if host != nil {
		return []msg.Service{{Host: r.joinDomain(host.Name), TTL: defaultTTL}}, nil
	}

	// 3. try guests table
	guest := models.GuestnetworkManager.GetGuestByAddress(ip)
	if guest != nil {
		return []msg.Service{{Host: r.joinDomain(guest.Name), TTL: defaultTTL}}, nil
	}
	return nil, errNoItems
}

func (r *SRegionDNS) joinDomain(name string) string {
	return strings.Join([]string{name, r.PrimaryZone}, ".")
}
