package dns

import (
	"strings"

	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"

	"github.com/yunionio/log"
)

type recordRequest struct {
	state      request.Request
	domainSegs []string
	projectId  string
	guestInfo  *SGuestInfo
}

func parseRequest(state request.Request) (r *recordRequest, err error) {
	base, _ := dnsutil.TrimZone(state.Name(), state.Zone)
	segs := dns.SplitDomainName(base)
	r = &recordRequest{
		state:      state,
		domainSegs: segs,
	}
	srcIP := r.SrcIP4()
	guestInfo := NewGuestInfoByAddress(srcIP)
	r.guestInfo = guestInfo
	return
}

func (r recordRequest) Name() string {
	//fullName, _ := dnsutil.TrimZone(r.state.Name(), "")
	name := r.state.Name()
	log.Errorf("==name: %q", name)
	name = strings.TrimSuffix(name, ".")
	return name
}

func (r recordRequest) GuestName() string {
	seps := strings.Split(r.Name(), ".")
	if len(seps) == 0 {
		return ""
	}
	return seps[0]
}

func (r recordRequest) Type() string {
	return DNSTypeMap[r.state.QType()]
}

func (r recordRequest) SrcIP4() string {
	ip := r.state.IP()
	log.Debugf("Source ip: %q, guestName: %q", ip, r.GuestName())
	return ip
}

func (r recordRequest) ProjectId() string {
	if r.guestInfo == nil {
		return ""
	}
	return r.guestInfo.GetProjectId()
}

func (r recordRequest) IsExitOnly() bool {
	if r.guestInfo == nil {
		return false
	}
	return r.guestInfo.IsExitOnly()
}
