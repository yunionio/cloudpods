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

package lbagent

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapis "yunion.io/x/onecloud/pkg/apis/compute"
	agentmodels "yunion.io/x/onecloud/pkg/lbagent/models"
	"yunion.io/x/onecloud/pkg/util/iproute2"
)

const (
	ErrOvnService = errors.Error("ovn controller")
	ErrOvnConfig  = errors.Error("ovn controller configuration")
)

type empty struct{}

type iptRule struct {
	rule    string
	comment string

	// computed
	key string
}

type Bar struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

func NewBar() *Bar {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	b := &Bar{
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
	}
	return b
}

func (b *Bar) Init() {
	b.wg.Add(1)
}

func (b *Bar) Done() {
	b.wg.Done()
}

func (b *Bar) Cancelled() <-chan struct{} {
	return b.ctx.Done()
}

func (b *Bar) Cancel() {
	b.cancel()
	b.wg.Wait()
}

type OvnHost struct {
	lb *agentmodels.Loadbalancer

	ovnBridge string

	macaddr string
	ipaddr  string
	masklen int
	gateway string

	vethAddrInner string
	vethAddrOuter string
	addrIdxInner  uint16
	addrIdxOuter  uint16

	vethAddrInnerPortMap map[string]int // key backendId
	portMap              map[int]empty  // key port
	portIdx              int

	Bar       *Bar
	refreshCh chan *ovnHostRefreshCmd

	ns netns.NsHandle
}

type ovnHostRefreshCmd struct {
	lb   *agentmodels.Loadbalancer
	done chan error
}

func newOvnHost(bridge string) *OvnHost {
	ovnHost := &OvnHost{
		vethAddrInnerPortMap: map[string]int{},
		portMap:              map[int]empty{},
		Bar:                  NewBar(),
		refreshCh:            make(chan *ovnHostRefreshCmd),
		ns:                   -1,

		ovnBridge: bridge,
	}
	return ovnHost
}

func (ovnHost *OvnHost) SetAddrIdx(inner, outer uint16) {
	ovnHost.addrIdxInner = inner
	ovnHost.addrIdxOuter = outer
	ovnHost.vethAddrInner = fmt.Sprintf("169.254.%d.%d", (inner>>8)&0xff, inner&0xff)
	ovnHost.vethAddrOuter = fmt.Sprintf("169.254.%d.%d", (outer>>8)&0xff, outer&0xff)
}

func (ovnHost *OvnHost) Start(ctx context.Context) {
	ovnHost.Bar.Init()
	defer ovnHost.Bar.Done()

	tick := time.NewTicker(11 * time.Second)
	for {
		select {
		case <-tick.C:
			ovnHost.run(ctx)
		case cmd := <-ovnHost.refreshCh:
			err := ovnHost.refresh(ctx, cmd.lb)
			ovnHost.run(ctx)
			cmd.done <- err
		case <-ovnHost.Bar.Cancelled():
			ovnHost.stop()
			return
		case <-ctx.Done():
			return
		}
	}
}

// stop releases resources, cleans up modifications.  It's reentrant
func (ovnHost *OvnHost) stop() {
	ovnHost.cleanUp()
}

func (ovnHost *OvnHost) cleanUp() {
	ovnHost.cleanUpNetns()
	ovnHost.cleanUpIface()
}

func (ovnHost *OvnHost) cleanUpIface() {
	ovnHost.cleanUpBridge()
	ovnHost.cleanUpVethPair()
}

func (ovnHost *OvnHost) cleanUpNetns() {
	netns.DeleteNamed(ovnHost.netnsName())
	if int(ovnHost.ns) >= 0 {
		ovnHost.ns.Close()
	}
}

func (ovnHost *OvnHost) cleanUpBridge() {
	bridge := ovnHost.ovnBridge
	_, peer0 := ovnHost.ovnPairName()
	args := []string{
		"ovs-vsctl",
		"--", "--if-exists", "del-port", bridge, peer0,
	}
	var cancelFunc context.CancelFunc
	ctx := context.Background()
	ctx, cancelFunc = context.WithTimeout(ctx, time.Second*7)
	defer cancelFunc()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	_, err := cmd.Output()
	if err != nil {
		err = errors.Wrapf(err, "cleanup %q: %s", bridge, strings.Join(args, " "))
		log.Errorf("%v", err)
	}
}

func (ovnHost *OvnHost) cleanUpVethPair() {
	_, peer0 := ovnHost.ovnPairName()
	_, peer1 := ovnHost.haproxyPairName()
	for _, name := range []string{peer0, peer1} {
		link, err := netlink.LinkByName(name)
		if err != nil {
			if _, ok := err.(netlink.LinkNotFoundError); ok {
				continue
			}
			err = errors.Wrapf(err, "ovnHost.stop: LinkByName(%q)", name)
			log.Errorf("%v", err)
			continue
		}
		if err := netlink.LinkDel(link); err != nil {
			err = errors.Wrapf(err, "ovnHost.stop: LinkDel(%q)", name)
			log.Errorf("%v", err)
		}
	}
}

func (ovnHost *OvnHost) Stop() {
	ovnHost.Bar.Cancel()
}

func (ovnHost *OvnHost) netnsName() string {
	return fmt.Sprintf("lb-%s", ovnHost.lb.Id)
}

func (ovnHost *OvnHost) lspName() string {
	return fmt.Sprintf("iface/lb/%s", ovnHost.lb.Id)
}

func (ovnHost *OvnHost) vethPairName(p string) (string, string) {
	// - index 0 is for the ovn host side
	// - index 1 is for the outside side (initial namespace)
	pref := fmt.Sprintf("%s%s", p, ovnHost.lb.Id[:15-len(p)-1])
	return pref + "0", pref + "1"
}

func (ovnHost *OvnHost) ovnPairName() (string, string) {
	// o is for connecting to ovn virtual network
	return ovnHost.vethPairName("o")
}

func (ovnHost *OvnHost) haproxyPairName() (string, string) {
	// h is for use with haproxy
	return ovnHost.vethPairName("h")
}

func (ovnHost *OvnHost) run(ctx context.Context) {
	defer func() {
		if msg := recover(); msg != nil {
			log.Errorf("ovn host for loadbalancer %s: %s", ovnHost.lb.Id, msg)
		}
	}()

	// - peer is the one in initial net namespace
	me0, peer0 := ovnHost.ovnPairName()
	me1, peer1 := ovnHost.haproxyPairName()
	ovnHost.ensureNs(ctx)
	ovnHost.ensureCleanNs(ctx, me0, peer0)
	ovnHost.ensureCleanNs(ctx, me1, peer1)
	ovnHost.ensureVethPair(ctx, me0, peer0)
	ovnHost.ensureVethPair(ctx, me1, peer1)

	if err := ovnHost.nsRun(ctx, func(ctx context.Context) error {
		// ovn side
		if err := iproute2.NewLink(me0).Address(ovnHost.macaddr).Up().Err(); err != nil {
			panic(errors.Wrapf(err, "set link %q up", me0))
		}
		if err := iproute2.NewAddress(me0, fmt.Sprintf("%s/%d", ovnHost.ipaddr, ovnHost.masklen)).Exact().Err(); err != nil {
			panic(errors.Wrapf(err, "set address of link %q", me0))
		}
		if err := iproute2.NewRoute(me0).AddByCidr("0.0.0.0/0", ovnHost.gateway).Err(); err != nil {
			panic(errors.Wrapf(err, "add route to inner addr %q", ovnHost.vethAddrInner))
		}

		// haproxy side
		if err := iproute2.NewLink(me1).Up().Err(); err != nil {
			panic(errors.Wrapf(err, "set link %q up", me0))
		}
		if err := iproute2.NewAddress(me1, ovnHost.vethAddrInner+"/16").Exact().Err(); err != nil {
			panic(errors.Wrapf(err, "set address of link %q", me0))
		}

		var (
			// NOTE: make sure arguments do not contain blank chars
			// or such things to avoid bad things like injection
			preRules  []iptRule
			postRules []iptRule
		)
		// iptables SNAT rule
		postRules = append(postRules, iptRule{
			rule:    fmt.Sprintf("-o %s -j SNAT --to-source %s", me0, ovnHost.ipaddr),
			comment: "snat-ovn",
		})
		postRules = append(postRules, iptRule{
			rule:    fmt.Sprintf("-o %s -j SNAT --to-source %s", me1, ovnHost.vethAddrInner),
			comment: "snat-haproxy",
		})
		// iptables listener rule
		for _, listener := range ovnHost.lb.Listeners {
			var proto string
			switch listener.ListenerType {
			case computeapis.LB_LISTENER_TYPE_TCP,
				computeapis.LB_LISTENER_TYPE_HTTP,
				computeapis.LB_LISTENER_TYPE_HTTPS:
				proto = "tcp"
			case computeapis.LB_LISTENER_TYPE_UDP:
				proto = "udp"
			default:
				panic(errors.Errorf("listener %s(%s) protocol: %q", listener.Name, listener.Id, listener.ListenerType))
			}
			preRules = append(preRules, iptRule{
				rule: fmt.Sprintf("-i %s -d %s -p %s --dport %d -j DNAT --to-destination %s:%d",
					me0, ovnHost.ipaddr, proto, listener.ListenerPort, ovnHost.vethAddrOuter, listener.ListenerPort),
				comment: fmt.Sprintf("listener/%s/%s/%d", listener.Id, listener.ListenerType, listener.ListenerPort),
			})
		}
		// iptables backend rule
		for _, backendGroup := range ovnHost.lb.BackendGroups {
			for _, backend := range backendGroup.Backends {
				port, ok := ovnHost.vethAddrInnerPortMap[backend.Id]
				if !ok {
					log.Warningf("cannot find netns port for backend %s(%s) %s:%d",
						backend.Name, backend.Id, backend.Address, backend.Port)
					continue
				}
				for _, proto := range []string{"tcp", "udp"} {
					preRules = append(preRules, iptRule{
						rule: fmt.Sprintf("-i %s -d %s -p %s --dport %d -j DNAT --to-destination %s:%d",
							me1, ovnHost.vethAddrInner, proto, port, backend.Address, backend.Port),
						comment: fmt.Sprintf("backend/%s/%s/%s/%d", backend.Id, proto, backend.Address, backend.Port),
					})
				}
			}
		}
		ovnHost.ensureIptRules(ctx, "nat", "PREROUTING", preRules)
		ovnHost.ensureIptRules(ctx, "nat", "POSTROUTING", postRules)
		return nil
	}); err != nil {
		panic(errors.Wrap(err, "run in netns"))
	}

	{ // setup brvpc
		bridge := ovnHost.ovnBridge
		lsp := ovnHost.lspName()
		args := []string{
			"ovs-vsctl",
			"--", "--may-exist", "add-port", bridge, peer0,
			"--", "set", "Interface", peer0, "external_ids:iface-id=" + lsp,
		}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		_, err := cmd.Output()
		if err != nil {
			panic(errors.Wrap(err, strings.Join(args, " ")))
		}
	}

	// set peer links up
	for _, link := range []string{peer0, peer1} {
		if err := iproute2.NewLink(link).Up().Err(); err != nil {
			panic(errors.Wrapf(err, "set link %q up", link))
		}
	}

	// set address, route for peer1 link
	if peer1Addr, err := netlink.ParseAddr(ovnHost.vethAddrOuter + "/16"); err != nil {
		panic(errors.Wrapf(err, "parse addr %q for link %q", ovnHost.vethAddrOuter+"/16", peer1))
	} else {
		peer1Addr.Flags |= unix.IFA_F_NOPREFIXROUTE
		if err := iproute2.NewAddressEx(peer1, peer1Addr).Exact().Err(); err != nil {
			panic(errors.Wrapf(err, "set address of link %q", peer1))
		}
	}
	if err := iproute2.NewRoute(peer1).AddByCidr(ovnHost.vethAddrInner+"/32", "").Err(); err != nil {
		panic(errors.Wrapf(err, "add route to inner addr %q", ovnHost.vethAddrInner))
	}
}

func (ovnHost *OvnHost) ensureNs(ctx context.Context) {
	if int(ovnHost.ns) >= 0 {
		return
	}
	netnsName := ovnHost.netnsName()
	{
		p := path.Join("/var/run/netns", netnsName)
		if _, err := ioutil.ReadFile(p); err == nil {
			err := os.Remove(p)
			log.Warningf("removing possible leftover file: %s: %v", p, err)
		}
	}
	if ns, err := netns.GetFromName(netnsName); err == nil {
		ovnHost.ns = ns
		return
	}
	if err := ovnHost.nsRun_(ctx, func(ctx context.Context) error {
		var err error
		ovnHost.ns, err = netns.NewNamed(netnsName)
		if err != nil {
			return errors.Wrapf(err, "new netns")
		}
		return nil
	}); err != nil {
		panic(err)
	}
}

func (ovnHost *OvnHost) ensureCleanNs(ctx context.Context, me, peer string) {
	var meOk, peerOk bool
	ovnHost.nsRun(ctx, func(ctx context.Context) error {
		_, err := netlink.LinkByName(me)
		if err == nil {
			meOk = true
		}
		return nil
	})
	_, err := netlink.LinkByName(peer)
	if err == nil {
		peerOk = true
	}
	if !meOk && peerOk {
		log.Warningf("clean up iface because %q %v, %q %v", me, meOk, peer, peerOk)
		ovnHost.cleanUpIface()
	}
}

func (ovnHost *OvnHost) ensureVethPair(ctx context.Context, me, peer string) {
	link, err := netlink.LinkByName(peer)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); !ok {
			panic(errors.Wrapf(err, "veth: LinkByName(%q)", peer))
		}
	} else if typ := link.Type(); typ != "veth" {
		panic(errors.Wrapf(err, "veth: LinkByName(%q) found but of type %s, not veth", peer, typ))
	} else {
		return
	}

	veth := &netlink.Veth{}
	veth.Name = me
	veth.PeerName = peer
	if err := netlink.LinkAdd(veth); err != nil {
		panic(errors.Wrapf(err, "add veth pair %q, %q", me, peer))
	}
	if err = netlink.LinkSetNsFd(veth, int(ovnHost.ns)); err != nil {
		panic(errors.Wrapf(err, "set ns of link %q", veth.Name))
	}
}

func (ovnHost *OvnHost) ensureIptRules(ctx context.Context, tbl, chain string, iptRules []iptRule) {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		panic(errors.Wrap(err, "ipt client"))
	}
	oldLines, err := ipt.List(tbl, chain)
	if err != nil {
		panic(errors.Wrapf(err, "ipt list tbl %q chain %q", tbl, chain))
	}
	for i := len(iptRules) - 1; i >= 0; i-- {
		iptRule := &iptRules[i]
		h := md5.Sum([]byte(iptRule.rule + iptRule.comment))
		key := base64.RawStdEncoding.EncodeToString(h[:])
		iptRule.key = key
		for j := len(oldLines) - 1; j >= 0; j-- {
			line := oldLines[j]
			if strings.Contains(line, key) {
				// found a match
				iptRules = append(iptRules[:i], iptRules[i+1:]...)
				oldLines = append(oldLines[:j], oldLines[j+1:]...)
				break
			}
		}
	}
	// cleanup old lines
	for _, line := range oldLines {
		if !strings.HasPrefix(line, "-A ") {
			continue
		}
		args := strings.Fields(line)
		if len(args) < 2 {
			continue
		}
		args = args[2:]
		for i, arg := range args {
			if len(arg) >= 2 && strings.HasPrefix(arg, `"`) && strings.HasSuffix(arg, `"`) {
				args[i] = arg[1 : len(arg)-1]
			}
		}
		if err := ipt.Delete(tbl, chain, args...); err != nil {
			panic(errors.Wrapf(err, "ipt delete %q %q: %q", tbl, chain, line))
		}
	}
	// add missing ones
	for i := range iptRules {
		iptRule := &iptRules[i]
		rule := fmt.Sprintf("-m comment --comment %s:%s ", iptRule.key, iptRule.comment)
		rule += iptRule.rule
		args := strings.Fields(rule)
		if err := ipt.Append(tbl, chain, args...); err != nil {
			panic(errors.Wrapf(err, "ipt append %q %q: %q", tbl, chain, rule))
		}
	}
}

func (ovnHost *OvnHost) nsRun(ctx context.Context, f func(ctx context.Context) error) error {
	return ovnHost.nsRun_(ctx, func(ctx context.Context) error {
		if err := netns.Set(ovnHost.ns); err != nil {
			return errors.Wrapf(err, "nsRun: set netns %s", ovnHost.ns)
		}
		return f(ctx)
	})
}

func (ovnHost *OvnHost) nsRun_(ctx context.Context, f func(ctx context.Context) error) error {
	var (
		wg  = &sync.WaitGroup{}
		err error
	)
	origNs, err := netns.Get()
	if err != nil {
		return errors.Wrap(err, "get current net ns")
	}
	defer origNs.Close()
	wg.Add(1)
	go func() {
		defer wg.Done()

		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		defer netns.Set(origNs)

		err = f(ctx)

	}()
	wg.Wait()
	return err
}

func (ovnHost *OvnHost) Refresh(ctx context.Context, lb *agentmodels.Loadbalancer) error {
	cmd := &ovnHostRefreshCmd{
		lb:   lb,
		done: make(chan error),
	}

	select {
	case ovnHost.refreshCh <- cmd:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-cmd.done:
		return err
	}

	// return nil
}

func (ovnHost *OvnHost) refresh(ctx context.Context, lb *agentmodels.Loadbalancer) error {
	// We may allow detch/attach network for loadbalancer in the future
	lbnet := lb.LoadbalancerNetwork
	if len(lbnet.MacAddr) == 0 {
		return errors.Errorf("empty LoadbalancerNetwork MacAddr for lb %s(%s)? mismatch region version???", lb.Name, lb.Id)
	}
	lb.ListenAddress = ovnHost.vethAddrOuter
	ovnHost.lb = lb
	ovnHost.macaddr = lbnet.MacAddr
	ovnHost.ipaddr = lbnet.IpAddr
	ovnHost.masklen = int(lbnet.Network.GuestIpMask)
	ovnHost.gateway = lbnet.Network.GuestGateway

	backendGroups := lb.BackendGroups
	backends := map[string]*agentmodels.LoadbalancerBackend{}
	for _, backendGroup := range backendGroups {
		for _, backend := range backendGroup.Backends {
			backendId := backend.Id
			backends[backendId] = backend
		}
	}

	// release first
	for backendId, port := range ovnHost.vethAddrInnerPortMap {
		if _, ok := backends[backendId]; !ok {
			delete(ovnHost.portMap, port)
			delete(ovnHost.vethAddrInnerPortMap, backendId)
		}
	}

	// allocate
	for _, backend := range backends {
		backendId := backend.Id
		if p, ok := ovnHost.vethAddrInnerPortMap[backendId]; ok {
			backend.ConnectAddress = ovnHost.vethAddrInner
			backend.ConnectPort = p
			continue
		}
		p := ovnHost.portIdx
		for {
			p += 1
			p &= 0xffff
			if p <= 1024 {
				p = 1025
			}
			if p == ovnHost.portIdx {
				return errors.Errorf("no available port for mapping: loadbalancer %s(%s) has too many backends",
					lb.Name, lb.Id)
			}
			if _, ok := ovnHost.portMap[p]; !ok {
				ovnHost.portMap[p] = empty{}
				ovnHost.vethAddrInnerPortMap[backendId] = p
				backend.ConnectAddress = ovnHost.vethAddrInner
				backend.ConnectPort = p
				break
			}
		}
		ovnHost.portIdx = p
	}
	return nil
}

type OvnWorker struct {
	opts *Options

	lbMap map[string]*OvnHost // key loadbalancerId

	addrMap map[uint16]*OvnHost // key x.x in 169.254.x.x
	addrIdx uint16

	Bar       *Bar
	refreshCh chan *ovnRefreshCmd
}

type ovnRefreshCmd struct {
	lbs  agentmodels.Loadbalancers
	done chan error
}

func NewOvnWorker(opts *Options) *OvnWorker {
	ovn := &OvnWorker{
		opts:      opts,
		lbMap:     map[string]*OvnHost{},
		addrMap:   map[uint16]*OvnHost{},
		Bar:       NewBar(),
		refreshCh: make(chan *ovnRefreshCmd),
	}
	return ovn
}

func (ovn *OvnWorker) Start(ctx context.Context) {
	log.Infof("ovn worker started")
	ovn.Bar.Init()
	defer ovn.Bar.Done()
	defer log.Infof("ovn worker bye!")
	for {
		select {
		case cmd := <-ovn.refreshCh:
			err := ovn.refresh(ctx, cmd.lbs)
			cmd.done <- err
		case <-ovn.Bar.Cancelled():
			log.Infof("ovn worker stop on cancel signal")
			ovn.stop()
			return
		case <-ctx.Done():
			return
		}
	}
}

func (ovn *OvnWorker) stop() {
	for _, o := range ovn.lbMap {
		o.Stop()
	}
}

func (ovn *OvnWorker) Stop() {
	ovn.Bar.Cancel()
}

func (ovn *OvnWorker) findTwoAddrIdx() (uint16, uint16, error) {
	// fetch current addresses
	sysAddrs := map[uint16]empty{}
	{
		ifaddrs, err := net.InterfaceAddrs()
		if err != nil {
			return 0, 0, errors.Wrap(err, "fetch current system unicast addrs")
		}
		for _, ifaddr := range ifaddrs {
			ipaddr, ok := ifaddr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipaddr.IP
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			if ip4[0] != 169 && ip4[1] != 254 {
				continue
			}
			i := uint16(ip4[2])<<8 | uint16(ip4[3])
			sysAddrs[i] = empty{}
		}
	}

	allocOne := func() (uint16, error) {
		i := ovn.addrIdx
		for {
			i += 1
			i &= 0xffff
			if i < 100 {
				i = 100
			}
			if i == ovn.addrIdx {
				return 0, errors.Error("169.254.x.x addrs run out")
			}
			if _, ok := sysAddrs[i]; ok {
				continue
			}
			if _, ok := ovn.addrMap[i]; !ok {
				break
			}
		}
		ovn.addrIdx = i
		return i, nil
	}
	inner, err := allocOne()
	if err != nil {
		return 0, 0, errors.Wrap(err, "inner address")
	}
	outer, err := allocOne()
	if err != nil {
		return 0, 0, errors.Wrap(err, "outer address")
	}
	return inner, outer, nil
}

func (ovn *OvnWorker) Refresh(ctx context.Context, lbs agentmodels.Loadbalancers) error {
	cmd := &ovnRefreshCmd{
		lbs:  lbs,
		done: make(chan error),
	}

	select {
	case ovn.refreshCh <- cmd:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-cmd.done:
		return err
	}

	// return nil
}

func (ovn *OvnWorker) refresh(ctx context.Context, lbs agentmodels.Loadbalancers) error {
	m := ovn.lbMap
	// release deleted
	for lbId, ovnHost := range m {
		if _, ok := lbs[lbId]; ok {
			continue
		}
		delete(m, lbId)
		delete(ovn.addrMap, ovnHost.addrIdxInner)
		delete(ovn.addrMap, ovnHost.addrIdxOuter)
		ovnHost.Stop()
	}

	// allocate
	for _, lb := range lbs {
		if lb.NetworkType != computeapis.LB_NETWORK_TYPE_VPC {
			continue
		}
		if len(lb.LoadbalancerNetwork.MacAddr) == 0 {
			log.Errorf("empty LoadbalancerNetwork MacAddr for lb %s(%s)? mismatch region version???", lb.Name, lb.Id)
			continue
		}
		ovnHost, ok := m[lb.Id]
		if !ok {
			inner, outer, err := ovn.findTwoAddrIdx()
			if err != nil {
				return errors.Wrapf(err, "find 169.254.x.x for lb %s(%s)", lb.Name, lb.Id)
			}
			ovnHost = newOvnHost(ovn.opts.OvnIntegrationBridge)
			ovnHost.SetAddrIdx(inner, outer)
			ovn.addrMap[inner] = ovnHost
			ovn.addrMap[outer] = ovnHost
			ovn.lbMap[lb.Id] = ovnHost
			go ovnHost.Start(ctx)
		}
		if err := ovnHost.Refresh(ctx, lb); err != nil {
			return err
		}
	}
	return nil
}
