package dns

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	//"os"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/etcd/msg"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/request"
	_ "github.com/go-sql-driver/mysql"
	//clog "github.com/coredns/coredns/plugin/log"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	//"github.com/coredns/coredns/plugin/pkg/replacer"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"
	ylog "github.com/yunionio/log"
	"github.com/yunionio/pkg/utils"
	"github.com/yunionio/sqlchemy"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

const (
	PluginName string = "yunion"
)

var (
	DNSTypeMap map[uint16]string = map[uint16]string{
		dns.TypeA:     "A",
		dns.TypeAAAA:  "AAAA",
		dns.TypeTXT:   "TXT",
		dns.TypeCNAME: "CNAME",
		dns.TypePTR:   "PTR",
		dns.TypeMX:    "MX",
		dns.TypeSRV:   "SRV",
		dns.TypeSOA:   "SOA",
		dns.TypeNS:    "NS",
	}
)

type SRegionDNS struct {
	Next          plugin.Handler
	Fall          fall.F
	Zones         []string
	Upstream      upstream.Upstream
	SqlConnection string
	K8sConfigFile string
}

func New() *SRegionDNS {
	r := new(SRegionDNS)
	return r
}

func (r *SRegionDNS) initDB(c *caddy.Controller) error {
	dialect, sqlStr, err := utils.TransSQLAchemyURL(r.SqlConnection)
	if err != nil {
		return err
	}
	dbConn, err := sql.Open(dialect, sqlStr)
	if err != nil {
		return err
	}
	sqlchemy.SetDB(dbConn)
	db.InitAllManagers()

	c.OnShutdown(func() error {
		r.CloseDB()
		return nil
	})
	return nil
}

func (r *SRegionDNS) initK8s(c *caddy.Controller) error {
	return nil
}

func (r *SRegionDNS) CloseDB() {
	sqlchemy.CloseDB()
}

func (r *SRegionDNS) ServeDNS(ctx context.Context, w dns.ResponseWriter, rmsg *dns.Msg) (int, error) {
	//rrw := dnstest.NewRecorder(w)
	//rep := replacer.New(r, rrw, corelog.CommonLogEmptyValue)
	//log.Infof("%v", rep.Replace(format))
	//fmt.Fprintln(output, rep.Replace(format))
	//count := models.DnsRecordManager.QueryDns("", "drone.yunion.io")
	var (
		records []dns.RR
		extra   []dns.RR
		err     error
	)

	opt := plugin.Options{}
	state := request.Request{W: w, Req: rmsg, Context: ctx}

	//isMyDomain := true
	zone := plugin.Zones(r.Zones).Matches(state.Name())
	if zone == "" {
		//isMyDomain = false
		return plugin.NextOrFailure(r.Name(), r.Next, ctx, w, rmsg)
	}

	switch state.QType() {
	case dns.TypeA:
		ylog.Debugf("===A question: %#v", state)
		records, err = plugin.A(r, zone, state, nil, opt)
	case dns.TypeAAAA:
		ylog.Debugf("===AAAA question: %#v", state)
		records, err = plugin.AAAA(r, zone, state, nil, opt)
	case dns.TypeTXT:
		ylog.Debugf("===TXT question: %#v", state)
		records, err = plugin.TXT(r, zone, state, opt)
	case dns.TypeCNAME:
		ylog.Debugf("===CNAME question: %#v", state)
		records, err = plugin.CNAME(r, zone, state, opt)
	case dns.TypePTR:
		ylog.Debugf("===PTR question: %#v", state)
		records, err = plugin.PTR(r, zone, state, opt)
	case dns.TypeMX:
		ylog.Debugf("===MX question: %#v", state)
		records, extra, err = plugin.MX(r, zone, state, opt)
	case dns.TypeSRV:
		ylog.Debugf("===SRV question: %#v", state)
		records, extra, err = plugin.SRV(r, zone, state, opt)
	case dns.TypeSOA:
		ylog.Debugf("===SOA question: %#v", state)
		records, err = plugin.SOA(r, zone, state, opt)
	case dns.TypeNS:
		ylog.Debugf("===NS question: %#v", state)
		if state.Name() == zone {
			records, extra, err = plugin.NS(r, zone, state, opt)
			break
		}
		fallthrough
	default:
		ylog.Infof("=== not processed state: %#v", state)
		// Do a fake A lookup, so we can distinguish between NODATA and NXDOMAIN
		_, err = plugin.A(r, zone, state, nil, opt)
	}

	if r.IsNameError(err) {
		if r.Fall.Through(state.Name()) {
			return plugin.NextOrFailure(r.Name(), r.Next, ctx, w, rmsg)
		}
		return plugin.BackendError(r, zone, dns.RcodeNameError, state, nil /* err */, opt)
	}
	if err != nil {
		//return dns.RcodeServerFailure, err
		return plugin.BackendError(r, zone, dns.RcodeServerFailure, state, err, opt)
	}

	if len(records) == 0 {
		return plugin.BackendError(r, zone, dns.RcodeSuccess, state, err, opt)
	}

	m := new(dns.Msg)
	m.SetReply(rmsg)
	m.Authoritative, m.RecursionAvailable = true, true
	m.Answer = append(m.Answer, records...)
	m.Extra = append(m.Extra, extra...)

	state.SizeAndDo(m)
	m, _ = state.Scrub(m)
	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

var (
	errNoItems = errors.New("no items found")
)

// Services implements the ServiceBackend interface
func (r *SRegionDNS) Services(state request.Request, exact bool, opt plugin.Options) (services []msg.Service, err error) {
	//services, err = r.Records(state, exact)
	//if err != nil {
	//return
	//}
	//services = msg.Group
	switch state.QType() {

	case dns.TypeTXT:
		t, _ := dnsutil.TrimZone(state.Name(), state.Zone)

		segs := dns.SplitDomainName(t)
		if len(segs) != 1 {
			return nil, fmt.Errorf("yunion region: TXT query can onlyu be for dns-version: %s", state.QName())
		}
		if segs[0] != "dns-version" {
			return nil, nil
		}
		svc := msg.Service{Text: "0.0.1", TTL: 28800, Key: msg.Path(state.QName(), "coredns")}
		return []msg.Service{svc}, nil

	case dns.TypeNS:
		ns := r.nsAddr()
		svc := msg.Service{Host: ns.A.String(), Key: msg.Path(state.QName(), "coredns")}
		return []msg.Service{svc}, nil
	}

	if state.QType() == dns.TypeA && isDefaultNS(state.Name(), state.Zone) {
		// If this is an A request for "ns.dns", respond with a "fake" record for coredns.
		// SOA records always use this hardcoded name
		ns := r.nsAddr()
		svc := msg.Service{Host: ns.A.String(), Key: msg.Path(state.QName(), "coredns")}
		return []msg.Service{svc}, nil
	}

	s, e := r.Records(state, false)

	// SRV is not yet implemented, so remove those records.
	if state.QType() != dns.TypeSRV {
		return s, e
	}

	internal := []msg.Service{}
	for _, svc := range s {
		if t, _ := svc.HostType(); t != dns.TypeCNAME {
			internal = append(internal, svc)
		}
	}
	return internal, e
}

// Reverse implements the ServiceBackend interface
func (r *SRegionDNS) Reverse(state request.Request, exact bool, opt plugin.Options) (services []msg.Service, err error) {
	return r.Services(state, exact, opt)
}

// Lookup implements the ServiceBackend interface
func (r *SRegionDNS) Lookup(state request.Request, name string, typ uint16) (*dns.Msg, error) {
	return r.Upstream.Lookup(state, name, typ)
}

// IsNameError implements the ServiceBackend interface
func (r *SRegionDNS) IsNameError(err error) bool {
	return err == errNoItems
}

// Records looks up records in region mysql
func (r *SRegionDNS) Records(state request.Request, exact bool) ([]msg.Service, error) {
	req, e := parseRequest(state)
	if e != nil {
		return nil, e
	}
	return r.findRecords(req)
}

func (r *SRegionDNS) getGuestIpWithName(req *recordRequest) []string {
	ips := []string{}
	name := req.GuestName()
	projectId := req.ProjectId()
	isExitOnly := req.IsExitOnly()
	ylog.Warningf("===========args: %q, %q, %v", projectId, name, isExitOnly)
	ips = models.GuestManager.GetIpInProjectWithName(projectId, name, isExitOnly)
	return ips
}

func (r *SRegionDNS) Name() string {
	return PluginName
}

type SGuestInfo struct {
	*models.SGuest
}

func (g *SGuestInfo) GetProjectId() string {
	return g.SGuest.ProjectId
}

func (g *SGuestInfo) GetGuestId() string {
	return g.SGuest.Id
}

func (g *SGuestInfo) IsExitOnly() bool {
	return g.SGuest.IsExitOnly()
}

func NewGuestInfoByAddress(address string) *SGuestInfo {
	guest := models.GuestnetworkManager.GetGuestByAddress(address)
	if guest == nil {
		return nil
	}
	return &SGuestInfo{SGuest: guest}
}

func (r *SRegionDNS) findRecords(req *recordRequest) (recs []msg.Service, err error) {
	isMyDomain := false
	zone := plugin.Zones(r.Zones).Matches(req.state.Name())
	if zone != "" {
		isMyDomain = true
	}
	//isPrivateAddr := false
	if isMyDomain {
		return r.findInternalRecords(req)
	}
	return r.findExternalRecords(req)
}

func (r *SRegionDNS) findInternalRecords(req *recordRequest) ([]msg.Service, error) {
	ylog.Debugf("=======findInternalRecords, srcip: %q, projectId: %q", req.SrcIP4(), req.ProjectId())
	if req.ProjectId() == "" {
		return nil, nil
	}
	// first, try dns records
	svcs, _ := r.findLocalRecords(req)
	if len(svcs) != 0 {
		return svcs, nil
	}
	// second, try guest table
	ips := r.getGuestIpWithName(req)
	return ips2DnsRecords(ips), nil
}

func (r *SRegionDNS) findExternalRecords(req *recordRequest) ([]msg.Service, error) {
	srcIP := req.SrcIP4()
	ylog.Debugf("Get client ip: %q", srcIP)
	return r.findLocalRecords(req)
}

func (r *SRegionDNS) findLocalRecords(req *recordRequest) (recs []msg.Service, err error) {
	ips := models.DnsRecordManager.QueryDnsIps(req.ProjectId(), req.Name(), req.Type())
	if len(ips) == 0 {
		err = errNoItems
		return
	}
	for _, ip := range ips {
		s := msg.Service{Host: ip.Addr, TTL: 5 * 60}
		recs = append(recs, s)
	}
	return
}

func ips2DnsRecords(ips []string) []msg.Service {
	recs := make([]msg.Service, 0)
	for _, ip := range ips {
		s := msg.Service{Host: ip, TTL: 5 * 60}
		recs = append(recs, s)
	}
	return recs
}
