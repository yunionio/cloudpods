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
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/etcd/msg"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/coredns/coredns/request"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
	_ "yunion.io/x/sqlchemy/backends"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	identity_api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	PluginName string = "yunion"

	// defaultTTL to apply to all answers
	defaultTTL           = 300
	defaultDbMaxOpenConn = 32
	defaultDbMaxIdleConn = 32
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
	AuthUrl       string
	AdminProject  string
	AdminUser     string
	AdminDomain   string
	AdminPassword string
	Region        string

	AdminProjectDomain string

	InCloudOnly bool

	// K8sSkip bool

	// K8sManager *k8s.SKubeClusterManager

	primaryZoneLabelCount int
}

func New() *SRegionDNS {
	r := &SRegionDNS{}
	return r
}

func (r *SRegionDNS) initDB(c *caddy.Controller) error {
	dialect, sqlStr, err := utils.TransSQLAchemyURL(r.SqlConnection)
	if err != nil {
		return err
	}
	sqlDb, err := sql.Open(dialect, sqlStr)
	if err != nil {
		return err
	}
	sqlDb.SetMaxOpenConns(defaultDbMaxOpenConn)
	sqlDb.SetMaxIdleConns(defaultDbMaxIdleConn)
	sqlchemy.SetDB(sqlDb)
	db.InitAllManagers()

	c.OnShutdown(func() error {
		sqlchemy.CloseDB()
		return nil
	})
	return nil
}

/*func (r *SRegionDNS) initK8s() {
	r.K8sManager = k8s.NewKubeClusterManager(r.Region, 30*time.Second)
	r.K8sManager.Start()
}*/

func (r *SRegionDNS) getAdminSession(ctx context.Context) *mcclient.ClientSession {
	return auth.GetAdminSession(ctx, r.Region)
}

func (r *SRegionDNS) initAuth() {
	if len(r.AdminDomain) == 0 {
		r.AdminDomain = identity_api.DEFAULT_DOMAIN_NAME
	}
	if len(r.AdminProjectDomain) == 0 {
		r.AdminProjectDomain = identity_api.DEFAULT_DOMAIN_NAME
	}
	authInfo := auth.NewAuthInfo(r.AuthUrl, r.AdminDomain, r.AdminUser, r.AdminPassword, r.AdminProject, r.AdminProjectDomain)
	auth.Init(authInfo, false, true, "", "")
}

func (r *SRegionDNS) ServeDNS(ctx context.Context, w dns.ResponseWriter, rmsg *dns.Msg) (int, error) {
	var (
		records []dns.RR
		extra   []dns.RR
		err     error
	)

	opt := plugin.Options{}
	state := request.Request{W: w, Req: rmsg, Context: ctx}
	zone := plugin.Zones(r.Zones).Matches(state.Name())
	switch state.QType() {
	case dns.TypeA:
		records, err = plugin.A(r, zone, state, nil, opt)
	case dns.TypeAAAA:
		records, err = plugin.AAAA(r, zone, state, nil, opt)
	case dns.TypeTXT:
		records, err = plugin.TXT(r, zone, state, opt)
	case dns.TypeCNAME:
		records, err = plugin.CNAME(r, zone, state, opt)
	case dns.TypePTR:
		records, err = plugin.PTR(r, zone, state, opt)
	case dns.TypeMX:
		records, extra, err = plugin.MX(r, zone, state, opt)
	case dns.TypeSRV:
		records, extra, err = plugin.SRV(r, zone, state, opt)
	case dns.TypeSOA:
		records, err = plugin.SOA(r, zone, state, opt)
	case dns.TypeNS:
		if state.Name() == zone {
			records, extra, err = plugin.NS(r, zone, state, opt)
			break
		}
		fallthrough
	default:
		log.Warningf("Not processed state: %#v", state)
		// Do a fake A lookup, so we can distinguish between NODATA and NXDOMAIN
		_, err = plugin.A(r, zone, state, nil, opt)
	}

	if err == errCallNext {
		if r.Fall.Through(state.Name()) {
			return plugin.NextOrFailure(r.Name(), r.Next, ctx, w, rmsg)
		}
		return plugin.BackendError(r, zone, dns.RcodeNameError, state, nil /* err */, opt)
	} else if err == errRefused {
		return plugin.BackendError(r, zone, dns.RcodeRefused, state, err, opt)
	} else if err == errNotFound {
		return plugin.BackendError(r, zone, dns.RcodeNameError, state, err, opt)
	}

	if len(records) == 0 {
		return plugin.BackendError(r, zone, dns.RcodeNameError, state, err, opt)
	}

	m := new(dns.Msg)
	m.SetReply(rmsg)
	m.Authoritative, m.RecursionAvailable = true, true
	m.Answer = append(m.Answer, records...)
	m.Extra = append(m.Extra, extra...)

	state.SizeAndDo(m)
	m = state.Scrub(m)
	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

var (
	errRefused  = errors.New("refused the query")
	errNotFound = errors.New("not found")
	errCallNext = errors.New("continue to next")
)

// Services implements the ServiceBackend interface
func (r *SRegionDNS) Services(state request.Request, exact bool, opt plugin.Options) ([]msg.Service, error) {
	var services []msg.Service
	var err error

	defer func() {
		if len(services) == 0 {
			log.Infof(`%s:%s %s - %d "%s %s empty response"`, state.RemoteAddr(), state.Port(), state.Proto(), state.Len(), state.Type(), state.Name())
			return
		}
		for _, service := range services {
			log.Infof(`%s:%s %s - %d "%s IN %s %s"`, state.RemoteAddr(), state.Port(), state.Proto(), state.Len(), state.Type(), state.Name(), jsonutils.Marshal(service).String())
		}
	}()

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
		services = []msg.Service{svc}
		return services, nil
	case dns.TypeNS:
		ns := r.nsAddr()
		svc := msg.Service{Host: ns.A.String(), Key: msg.Path(state.QName(), "coredns")}
		services = []msg.Service{svc}
		return services, nil
	}

	if state.QType() == dns.TypeA && isDefaultNS(state.Name(), state.Zone) {
		// If this is an A request for "ns.dns", respond with a "fake" record for coredns.
		// SOA records always use this hardcoded name
		ns := r.nsAddr()
		svc := msg.Service{Host: ns.A.String(), Key: msg.Path(state.QName(), "coredns")}
		services = []msg.Service{svc}
		return services, nil
	}

	if _, ok := DNSTypeMap[state.QType()]; !ok {
		return nil, errRefused
	}

	services, err = r.Records(state, false)
	if err != nil {
		// log.Errorf("Records %s fail: %s", state.Name(), err)
		return nil, err
	}
	return services, nil
}

// Lookup implements the ServiceBackend interface
func (r *SRegionDNS) Lookup(state request.Request, name string, typ uint16) (*dns.Msg, error) {
	return r.Upstream.Lookup(state, name, typ)
}

// IsNameError implements the ServiceBackend interface
func (r *SRegionDNS) IsNameError(err error) bool {
	return err == errCallNext
}

// Records looks up records in region mysql
func (r *SRegionDNS) Records(state request.Request, exact bool) ([]msg.Service, error) {
	req := parseRequest(state)

	if r.InCloudOnly && !req.srcInCloud {
		// deny external request
		return nil, errRefused
	}
	return r.findRecords(req)
}

func (r *SRegionDNS) getHostIpWithName(req *recordRequest) string {
	if req.Type() != "A" {
		return ""
	}
	name := req.QueryName()
	name = strings.TrimSuffix(name, ".")
	ctx := context.Background()
	host, _ := models.HostManager.FetchByName(ctx, nil, name)
	if host == nil {
		return ""
	}
	ip := host.(*models.SHost).AccessIp
	return ip
}

func (r *SRegionDNS) getGuestIpsWithName(req *recordRequest) []string {
	ips := []string{}
	name := req.QueryName()
	projectId := req.ProjectId()
	wantOnlyExit := false
	if req.Type() == "A" {
		ip4s := models.GuestManager.GetIpsInProjectWithName(projectId, name, wantOnlyExit, api.AddressTypeIPv4)
		if len(ip4s) > 0 {
			ips = append(ips, ip4s...)
		}
	}
	if req.Type() == "AAAA" {
		ip6s := models.GuestManager.GetIpsInProjectWithName(projectId, name, wantOnlyExit, api.AddressTypeIPv6)
		if len(ip6s) > 0 {
			ips = append(ips, ip6s...)
		}
	}
	return ips
}

/*func getK8sServiceBackends(cli *kubernetes.Clientset, req *recordRequest) ([]string, error) {
	queryInfo := req.GetK8sQueryInfo()
	pods, err := getK8sServicePods(cli, queryInfo.Namespace, queryInfo.ServiceName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = nil
		}
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
}*/

/*func (r *SRegionDNS) getK8sClient() (*kubernetes.Clientset, error) {
	return r.K8sManager.GetK8sClient()
}*/

/*func getK8sServicePods(cli *kubernetes.Clientset, namespace, name string) ([]v1.Pod, error) {
	svc, err := cli.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	labelSelector := labels.SelectorFromSet(svc.Spec.Selector)
	pods, err := cli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector.String(),
		FieldSelector: fields.Everything().String(),
	})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}*/

func (r *SRegionDNS) Name() string {
	return PluginName
}

func (r *SRegionDNS) queryLocalDnsRecords(req *recordRequest) []msg.Service {
	var (
		projId = req.ProjectId()
		getTtl = func(ttl int64) uint32 {
			if ttl == 0 {
				return defaultTTL
			}
			return uint32(ttl)
		}
	)
	recs := make([]msg.Service, 0)
	records, err := models.DnsRecordManager.QueryDns(projId, req.Name(), req.Type())
	if err != nil {
		log.Errorf("QueryDns %s %s error: %v", req.Type(), req.Name(), err)
		return nil
	}

	if req.IsSRV() {
		for i := range records {
			rec := records[i]
			// priority weight port host
			parts := strings.SplitN(rec.DnsValue, " ", 4)
			if len(parts) != 4 {
				log.Errorf("Invalid SRV records: %q", rec.DnsValue)
				return nil
			}
			_priority, _weight, _port, host := parts[0], parts[1], parts[2], parts[3]
			priority, err := strconv.Atoi(_priority)
			if err != nil {
				log.Errorf("SRV: invalid priority: %s", _priority)
				return nil
			}
			weight, err := strconv.Atoi(_weight)
			if err != nil {
				log.Errorf("SRV: invalid weight: %s", _weight)
				return nil
			}
			port, err := strconv.Atoi(_port)
			if err != nil {
				log.Errorf("SRV: invalid port: %s", _port)
				return nil
			}
			recs = append(recs, msg.Service{
				Host:     host,
				Port:     port,
				Weight:   weight,
				Priority: priority,
				TTL:      getTtl(rec.TTL),
			})
		}
	} else {
		for i := range records {
			rec := records[i]
			recs = append(recs, msg.Service{
				Host: rec.DnsValue,
				TTL:  getTtl(rec.TTL),
			})
		}
	}

	return recs
}

func (r *SRegionDNS) isMyDomain(req *recordRequest) bool {
	if r.PrimaryZone == "" {
		return false
	}
	qname := req.state.Name()
	qnameLabelCount := dns.CountLabel(qname)
	if qnameLabelCount <= r.primaryZoneLabelCount {
		return false
	}
	matched := dns.CompareDomainName(r.PrimaryZone, qname)
	if matched == r.primaryZoneLabelCount {
		return true
	}
	return false
}

func (r *SRegionDNS) findRecords(req *recordRequest) ([]msg.Service, error) {
	// 1. try local dns records table
	if !r.isMyDomain(req) {
		rrs := r.queryLocalDnsRecords(req)
		if len(rrs) > 0 {
			return rrs, nil
		}
	}

	isPlainName := req.IsPlainName()
	isMyDomain := r.isMyDomain(req)

	if isPlainName {
		isCloudIp := req.SrcInCloud()
		if isCloudIp {
			ips := r.findInternalRecordIps(req)
			if len(ips) > 0 {
				return ips2DnsRecords(ips), nil
			} else {
				return nil, errNotFound
			}
		} else {
			return nil, errRefused
		}
	} else if isMyDomain {
		ips := r.findInternalRecordIps(req)
		if len(ips) > 0 {
			return ips2DnsRecords(ips), nil
		} else {
			return nil, errNotFound
		}
	} else {
		return nil, errCallNext
	}
}

func (r *SRegionDNS) findInternalRecordIps(req *recordRequest) []string {
	{
		// 1. try guest table
		ips := r.getGuestIpsWithName(req)
		if len(ips) > 0 {
			return ips
		}
	}

	{
		// 2. try host table
		ip := r.getHostIpWithName(req)
		if len(ip) > 0 {
			return []string{ip}
		}
	}

	/*if !r.K8sSkip {
		k8sCli, err := r.getK8sClient()
		if err != nil {
			log.Warningf("Get k8s client error: %v, skip it.", err)
			return nil
		}
		// 3. try k8s service backends
		ips, err := getK8sServiceBackends(k8sCli, req)
		if err != nil {
			log.Errorf("Get k8s service backends error: %v", err)
		}
		return ips
	}*/

	return nil
}

func ips2DnsRecords(ips []string) []msg.Service {
	recs := make([]msg.Service, 0)
	for _, ip := range ips {
		s := msg.Service{Host: ip, TTL: defaultTTL}
		recs = append(recs, s)
	}
	return recs
}
