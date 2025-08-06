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

package hostdhcp

import (
	"fmt"
	"net"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/icmp6"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

type sRARequest struct {
	solicitation *icmp6.SRouterSolicitation
	attempts     int
	succAttempts int
}

func (s *SGuestDHCP6Server) handleRouterSolicitation(msg *icmp6.SRouterSolicitation) error {
	// solicitation request from guest
	var conf = s.getConfig(msg.SrcMac)
	if conf != nil && conf.ClientIP6 != nil && conf.Gateway6 != nil {
		req := &sRARequest{
			solicitation: msg,
			attempts:     1,
		}
		succ, err := s.sendRouterAdvertisement(msg)
		if err != nil {
			return errors.Wrapf(err, "sendRouterAdvertisement")
		}
		if succ {
			req.succAttempts++
		}
		s.raReqCh <- req
	}
	return nil
}

func (s *SGuestDHCP6Server) handleRouterAdvertisement(msg *icmp6.SRouterAdvertisement) error {
	// periodic router advertisement from local router
	return nil
}

func (s *SGuestDHCP6Server) handleNeighborSolicitation(msg *icmp6.SNeighborSolicitation) error {
	return nil
}

func (s *SGuestDHCP6Server) handleNeighborAdvertisement(msg *icmp6.SNeighborAdvertisement) error {
	if msg.IsSolicited {
		//log.Infof("Save mac %s for gw IP %s", msg.SrcMac, msg.TargetAddr.String())
		s.gwMacCache.Set(msg.TargetAddr.String(), msg.SrcMac)
	}
	return nil
}

func (s *SGuestDHCP6Server) requestGatewayMac(gwIP net.IP, vlanId uint16) {
	// Solicited-Node Multicast Address, FF02::1:FF00:0/104, 33:33:ff:00:00:00
	destIP := netutils2.IP2SolicitMcastIP(gwIP)
	destMac := netutils2.IP2SolicitMcastMac(gwIP)

	ns := &icmp6.SNeighborSolicitation{
		SBaseICMP6Message: icmp6.SBaseICMP6Message{
			SrcMac: s.ifaceDev.GetHardwareAddr(),
			SrcIP:  net.ParseIP(s.ifaceDev.Addr6LinkLocal),
			DstMac: destMac,
			DstIP:  destIP,
			Vlan:   vlanId,
		},
		TargetAddr: gwIP,
	}

	bytes, err := icmp6.EncodePacket(ns)
	if err != nil {
		log.Errorf("Encode NeighborSolicitation error: %v", err)
		return
	}

	err = s.conn.SendRaw(bytes, destMac)
	if err != nil {
		log.Errorf("Send RouterAdvertisement error: %v", err)
		return
	}
}

func (s *SGuestDHCP6Server) sendRouterAdvertisement(solicitation *icmp6.SRouterSolicitation) (bool, error) {
	var conf = s.getConfig(solicitation.SrcMac)
	if conf != nil && conf.ClientIP6 != nil && conf.Gateway6 != nil {
		gwMacObj := s.gwMacCache.AtomicGet(conf.Gateway6.String())
		if gwMacObj == nil {
			log.Debugf("No mac for gw IP %s, request it", conf.Gateway6.String())
			s.requestGatewayMac(conf.Gateway6, conf.VlanId)
			return false, nil
		}

		gwMac := gwMacObj.(net.HardwareAddr)

		pref := icmp6.PreferenceMedium
		//if conf.IsDefaultGW {
		//	pref = icmp6.PreferenceHigh
		//}

		_, ipnet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", conf.ClientIP6, conf.PrefixLen6))
		if err != nil {
			return false, errors.Wrapf(err, "ParseCIDR %s/%d", conf.ClientIP6, conf.PrefixLen6)
		}

		srcIpAddr, _ := netutils.Mac2LinkLocal(gwMac.String())
		gwLinkLocalIP := srcIpAddr.ToIP()
		ra := &icmp6.SRouterAdvertisement{
			SBaseICMP6Message: icmp6.SBaseICMP6Message{
				SrcMac: gwMac,
				// https://www.rfc-editor.org/rfc/rfc4861.html#page-18
				// !!! MUST be the link-local address assigned to the interface from which this message is sent.
				SrcIP:  gwLinkLocalIP,
				DstMac: solicitation.SrcMac,
				DstIP:  net.ParseIP("ff02::1"),
			},
			CurHopLimit:    64,
			IsManaged:      true,
			IsOther:        true,
			IsHomeAgent:    false,
			Preference:     pref,
			RouterLifetime: 9000,
			ReachableTime:  0,
			RetransTimer:   0,
			MTU:            uint32(conf.MTU),
			PrefixInfo: []icmp6.SPrefixInfoOption{
				{
					IsOnlink:          true,
					IsAutoconf:        false,
					Prefix:            ipnet.IP,
					PrefixLen:         conf.PrefixLen6,
					ValidLifetime:     4500,
					PreferredLifetime: 2250,
				},
			},
		}
		for i := range conf.Routes6 {
			route := conf.Routes6[i]
			if route.Gateway.String() == "::" {
				// on-link routes
				ra.PrefixInfo = append(ra.PrefixInfo, icmp6.SPrefixInfoOption{
					IsOnlink:          true,
					IsAutoconf:        false,
					Prefix:            route.Prefix,
					PrefixLen:         route.PrefixLen,
					ValidLifetime:     4500,
					PreferredLifetime: 2250,
				})
			} else if route.Gateway.String() != conf.Gateway6.String() {
				// routes forwarded by router
				ra.RouteInfo = append(ra.RouteInfo, icmp6.SRouteInfoOption{
					RouteLifetime: 9000,
					Prefix:        route.Prefix,
					PrefixLen:     route.PrefixLen,
					Preference:    pref,
				})
			}
		}

		bytes, err := icmp6.EncodePacket(ra)
		if err != nil {
			log.Errorf("Encode RouterAdvertisement error: %v", err)
			return false, errors.Wrapf(err, "EncodePacket")
		}

		err = s.conn.SendRaw(bytes, solicitation.SrcMac)
		if err != nil {
			log.Errorf("Send RouterAdvertisement error: %v", err)
		}
	}
	return true, nil
}

func (s *SGuestDHCP6Server) stopRAServer() {
	close(s.raExitCh)
	close(s.raReqCh)
}

func (s *SGuestDHCP6Server) startRAServer() {
	// a tiny RA server
	stop := false
	for !stop {
		select {
		case <-s.raExitCh:
			stop = true
		case raReq := <-s.raReqCh:
			// handle RA request
			s.raReqQueue = append(s.raReqQueue, raReq)
		case <-time.After(time.Second * time.Duration(options.HostOptions.Dhcp6RouterAdvertisementIntervalSecs)):
			// send RA
			// log.Infof("timeout, to announce RA %d requests", len(s.raReqQueue))
			if len(s.raReqQueue) > 0 {
				raReqQueue := s.raReqQueue
				s.raReqQueue = make([]*sRARequest, 0)

				for i := range raReqQueue {
					raReq := raReqQueue[i]
					raReq.attempts++
					succ, err := s.sendRouterAdvertisement(raReq.solicitation)
					if err != nil {
						log.Errorf("sendRouterAdvertisement error: %v", err)
						continue
					}
					if succ {
						raReq.succAttempts++
					}
					if raReq.succAttempts < options.HostOptions.Dhcp6RouterAdvertisementAttempts && raReq.attempts < 2*options.HostOptions.Dhcp6RouterAdvertisementAttempts {
						s.raReqCh <- raReq
					}
				}
			}
		}
	}
}
