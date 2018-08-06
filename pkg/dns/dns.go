package dns

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/etcd/msg"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/coredns/coredns/request"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	ylog "github.com/yunionio/log"
	"github.com/yunionio/pkg/utils"
	"github.com/yunionio/sqlchemy"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/onecloud/pkg/util/k8s"
)

const (
	PluginName string = "yunion"

	// defaultTTL to apply to all answers
	defaultTTL = 10
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
	PrimaryZone   string
	Upstream      upstream.Upstream
	SqlConnection string
	K8sConfigFile string
	K8sClient     *kubernetes.Clientset
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

func (r *SRegionDNS) initK8s(c *caddy.Controller) {
	cli, err := k8s.NewClientByFile(r.K8sConfigFile, nil)
	if err != nil {
		ylog.Errorf("Init kubernetes client error: %v", err)
		return
	}
	r.K8sClient = cli
	pods, err := cli.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		ylog.Errorf("Get all pods in kubernetes cluster error: %v", err)
		return
	}
	ylog.Infof("Init k8s client success, %d pods in the cluster", len(pods.Items))
}

func (r *SRegionDNS) CloseDB() {
	sqlchemy.CloseDB()
}

func (r *SRegionDNS) ServeDNS(ctx context.Context, w dns.ResponseWriter, rmsg *dns.Msg) (int, error) {
	var (
		records []dns.RR
		extra   []dns.RR
		err     error
	)

	opt := plugin.Options{}
	state := request.Request{W: w, Req: rmsg, Context: ctx}

	//isMyDomain := true
	zone := plugin.Zones(r.Zones).Matches(state.Name())
	//if zone == "" {
	////isMyDomain = false
	//return plugin.NextOrFailure(r.Name(), r.Next, ctx, w, rmsg)
	//}

	switch state.QType() {
	case dns.TypeA:
		ylog.Debugf("A question: %#v", state)
		records, err = plugin.A(r, zone, state, nil, opt)
	case dns.TypeAAAA:
		ylog.Debugf("AAAA question: %#v", state)
		records, err = plugin.AAAA(r, zone, state, nil, opt)
	case dns.TypeTXT:
		ylog.Debugf("TXT question: %#v", state)
		records, err = plugin.TXT(r, zone, state, opt)
	case dns.TypeCNAME:
		ylog.Debugf("CNAME question: %#v", state)
		records, err = plugin.CNAME(r, zone, state, opt)
	case dns.TypePTR:
		ylog.Debugf("PTR question: %#v", state)
		records, err = plugin.PTR(r, zone, state, opt)
	case dns.TypeMX:
		ylog.Debugf("MX question: %#v", state)
		records, extra, err = plugin.MX(r, zone, state, opt)
	case dns.TypeSRV:
		ylog.Debugf("SRV question: %#v", state)
		records, extra, err = plugin.SRV(r, zone, state, opt)
	case dns.TypeSOA:
		ylog.Debugf("SOA question: %#v", state)
		records, err = plugin.SOA(r, zone, state, opt)
	case dns.TypeNS:
		ylog.Debugf("NS question: %#v", state)
		if state.Name() == zone {
			records, extra, err = plugin.NS(r, zone, state, opt)
			break
		}
		fallthrough
	default:
		ylog.Warningf("Not processed state: %#v", state)
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
	ylog.Debugf("Get records: %#v, error: %v", s, e)

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

func (r *SRegionDNS) getHostIpWithName(req *recordRequest) []string {
	name := req.QueryName()
	host, _ := models.HostManager.FetchByName("", name)
	if host == nil {
		return nil
	}
	ip := host.(*models.SHost).AccessIp
	if len(ip) == 0 {
		return nil
	}
	return []string{ip}
}

func (r *SRegionDNS) getGuestIpWithName(req *recordRequest) []string {
	ips := []string{}
	name := req.QueryName()
	projectId := req.ProjectId()
	isExitOnly := req.IsExitOnly()
	ips = models.GuestManager.GetIpInProjectWithName(projectId, name, isExitOnly)
	return ips
}

func (r *SRegionDNS) getK8sServiceBackends(req *recordRequest) ([]string, error) {
	queryInfo := req.GetK8sQueryInfo()
	pods, err := r.getK8sServicePods(queryInfo.Namespace, queryInfo.ServiceName)
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0)
	for _, pod := range pods {
		ip := pod.Status.PodIP
		if len(ip) != 0 {
			ips = append(ips, ip)
		}
	}
	return ips, nil
}

func (r *SRegionDNS) getK8sServicePods(namespace, name string) ([]v1.Pod, error) {
	cli := r.K8sClient
	svc, err := cli.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	labelSelector := labels.SelectorFromSet(svc.Spec.Selector)
	pods, err := cli.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: labelSelector.String(),
		FieldSelector: fields.Everything().String(),
	})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (r *SRegionDNS) Name() string {
	return PluginName
}

func (r *SRegionDNS) queryLocalDnsRecords(req *recordRequest) (recs []msg.Service, err error) {
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

func (r *SRegionDNS) IsCloudNetworkIp(req *recordRequest) bool {
	if req.network != nil {
		return true
	}
	return false
}

func (r *SRegionDNS) IsK8sClientReady() bool {
	return r.K8sClient != nil
}

func (r *SRegionDNS) findRecords(req *recordRequest) (recs []msg.Service, err error) {
	// 1. try local dns records table
	recs, err = r.queryLocalDnsRecords(req)
	if len(recs) != 0 {
		return
	}

	isMyDomain := false
	zone := plugin.Zones(r.Zones).Matches(req.state.Name())
	if zone != "" {
		isMyDomain = true
	}

	isCloudIp := r.IsCloudNetworkIp(req)

	// 2. not my domain and src ip not in cloud network table
	//    query from upstream
	if !isMyDomain && !isCloudIp {
		err = errNoItems
		return
	}

	// 3. internal query
	ips, err := r.findInternalRecordIps(req)
	return ips2DnsRecords(ips), err
}

func (r *SRegionDNS) findInternalRecordIps(req *recordRequest) ([]string, error) {
	// 1. try host table
	ip := r.getHostIpWithName(req)
	if len(ip) != 0 {
		return ip, nil
	}
	// 2. try guest table
	ip = r.getGuestIpWithName(req)
	if len(ip) != 0 {
		return ip, nil
	}

	if !r.IsK8sClientReady() {
		ylog.Warningf("K8s client not ready, skip it.")
		return nil, errNoItems
	}
	// 3. try k8s service backends
	ips, err := r.getK8sServiceBackends(req)
	if len(ips) != 0 {
		return ips, nil
	}
	if err != nil {
		ylog.Errorf("Get k8s service backends error: %v", err)
	}
	return nil, errNoItems
}

func ips2DnsRecords(ips []string) []msg.Service {
	recs := make([]msg.Service, 0)
	for _, ip := range ips {
		s := msg.Service{Host: ip, TTL: defaultTTL}
		recs = append(recs, s)
	}
	return recs
}
