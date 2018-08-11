package dns

import (
	"strings"

	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"

	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/compute/models"
)

type recordRequest struct {
	state      request.Request
	domainSegs []string
	guest      *models.SGuest
	host       *models.SHost
	network    *models.SNetwork
}

func parseRequest(state request.Request) (r *recordRequest, err error) {
	base, _ := dnsutil.TrimZone(state.Name(), state.Zone)
	segs := dns.SplitDomainName(base)
	r = &recordRequest{
		state:      state,
		domainSegs: segs,
	}
	srcIP := r.SrcIP4()
	r.guest = models.GuestnetworkManager.GetGuestByAddress(srcIP)
	r.host = models.HostnetworkManager.GetHostByAddress(srcIP)
	r.network, _ = models.NetworkManager.GetNetworkOfIP(srcIP, "", tristate.None)
	return
}

func (r recordRequest) Name() string {
	//fullName, _ := dnsutil.TrimZone(r.state.Name(), "")
	name := r.state.Name()
	name = strings.TrimSuffix(name, ".")
	return name
}

func (r recordRequest) QueryName() string {
	seps := strings.Split(r.Name(), ".")
	if len(seps) == 0 {
		return ""
	}
	return seps[0]
}

func (r recordRequest) Type() string {
	return DNSTypeMap[r.state.QType()]
}

func (r recordRequest) IsSRV() bool {
	return r.Type() == DNSTypeMap[dns.TypeSRV]
}

func (r recordRequest) SrcIP4() string {
	ip := r.state.IP()
	return ip
}

func (r recordRequest) ProjectId() string {
	if r.guest != nil {
		return r.guest.ProjectId
	}
	if r.network != nil {
		return r.network.ProjectId
	}
	return ""
}

func (r recordRequest) IsExitOnly() bool {
	if r.guest == nil {
		return false
	}
	return r.guest.IsExitOnly()
}

type K8sQueryInfo struct {
	ServiceName string
	Namespace   string
}

func (r recordRequest) GetK8sQueryInfo() K8sQueryInfo {
	parts := strings.SplitN(r.Name(), ".", 3)
	var svcName string
	var namespace string
	if len(parts) >= 2 {
		svcName = parts[0]
		namespace = parts[1]
	} else {
		svcName = parts[0]
		namespace = "default"
	}
	return K8sQueryInfo{
		ServiceName: svcName,
		Namespace:   namespace,
	}
}
