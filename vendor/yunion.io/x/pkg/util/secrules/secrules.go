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

package secrules

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
)

type TSecurityRuleDirection string

const (
	SecurityRuleIngress = TSecurityRuleDirection("in")
	SecurityRuleEgress  = TSecurityRuleDirection("out")
)

type TSecurityRuleAction string

const (
	SecurityRuleAllow = TSecurityRuleAction("allow")
	SecurityRuleDeny  = TSecurityRuleAction("deny")
)

type TSecurityRuleRelation string

const (
	RELATION_INDEPENDENT = TSecurityRuleRelation("INDEPENDT")
	RELATION_IDENTICAL   = TSecurityRuleRelation("IDENTICAL")
	RELATION_SUBSET      = TSecurityRuleRelation("SUBSET")
	RELATION_SUPERSET    = TSecurityRuleRelation("SUPERSET")
	RELATION_NEXT_AHEAD  = TSecurityRuleRelation("NEXT_AHEAD")
	RELATION_NEXT_AFTER  = TSecurityRuleRelation("NEXT_AFTER")
	RELATION_OVERLAP     = TSecurityRuleRelation("OVERLAP")
)

type SecurityRule struct {
	Priority    int // [1, 100]
	Action      TSecurityRuleAction
	IPNet       *net.IPNet
	Protocol    string
	Direction   TSecurityRuleDirection
	PortStart   int
	PortEnd     int
	Ports       []int
	Description string
}

const (
	DIR_IN  = "in"
	DIR_OUT = "out"
)
const SEG_ACTION = 0
const SEG_IP = 1
const SEG_PROTO = 2
const SEG_PORT = 3
const SEG_END = 4

// const ACTION_ALLOW = "allow"
// const ACTION_DENY = "deny"
const PROTO_ANY = "any"
const PROTO_TCP = "tcp"
const PROTO_UDP = "udp"
const PROTO_ICMP = "icmp"

// non-wild protocols
var protocolsSupported = []string{
	PROTO_TCP,
	PROTO_UDP,
	PROTO_ICMP,
}

var (
	ErrInvalidProtocolAny  = errors.New("invalid protocol any with port option")
	ErrInvalidProtocolICMP = errors.New("invalid protocol icmp with port option")
	ErrInvalidPriority     = errors.New("invalid priority")
	ErrInvalidDirection    = errors.New("invalid direction")
	ErrInvalidAction       = errors.New("invalid action")
	ErrInvalidNet          = errors.New("invalid net")
	ErrInvalidIPAddr       = errors.New("invalid ip address")
	ErrInvalidProtocol     = errors.New("invalid protocol")
	ErrInvalidPortRange    = errors.New("invalid port range")
	ErrInvalidPort         = errors.New("invalid port")
)

func parsePortString(ps string) (int, error) {
	p, err := strconv.ParseUint(ps, 10, 16)
	if err != nil || p == 0 {
		return 0, ErrInvalidPort
	}
	return int(p), nil
}

func MustParseSecurityRule(s string) *SecurityRule {
	r, err := ParseSecurityRule(s)
	if err != nil {
		msg := fmt.Sprintf("parse security rule %q: %v", s, err)
		panic(msg)
	}
	return r
}

func ParseSecurityRule(pattern string) (*SecurityRule, error) {
	rule := &SecurityRule{}
	for _, direction := range []TSecurityRuleDirection{SecurityRuleIngress, SecurityRuleEgress} {
		if len(pattern) > len(direction)+1 && pattern[:len(direction)+1] == string(direction)+":" {
			rule.Direction, pattern = direction, strings.Replace(pattern, string(direction)+":", "", -1)
			break
		}
	}
	if rule.Direction == "" {
		return nil, ErrInvalidDirection
	}
	status := SEG_ACTION
	data := strings.Split(strings.TrimSpace(pattern), " ")
	index, seg := 0, ""
	for status != SEG_END {
		seg = ""
		if len(data) >= index+1 {
			seg = data[index]
			index++
		}
		if status == SEG_ACTION {
			if seg == string(SecurityRuleAllow) || seg == string(SecurityRuleDeny) {
				if seg == string(SecurityRuleAllow) {
					rule.Action = SecurityRuleAllow
				} else {
					rule.Action = SecurityRuleDeny
				}
				status = SEG_IP
			} else {
				return nil, ErrInvalidAction
			}
		} else if status == SEG_IP {
			matched := rule.ParseCIDR(seg)
			if !matched {
				index--
			}
			status = SEG_PROTO
		} else if status == SEG_PROTO {
			if seg == PROTO_ANY || seg == PROTO_ICMP {
				status = SEG_END
			} else if seg == PROTO_TCP || seg == PROTO_UDP {
				status = SEG_PORT
			} else {
				return nil, ErrInvalidProtocol
			}
			rule.Protocol = seg
		} else if status == SEG_PORT {
			status = SEG_END
			if err := rule.ParsePorts(seg); err != nil {
				return nil, err
			}
			return rule, nil
		}
	}
	return rule, nil
}

func (rule *SecurityRule) ParseCIDR(cidr string) bool {
	if regutils.MatchCIDR(cidr) {
		_, rule.IPNet, _ = net.ParseCIDR(cidr)
		return true
	}
	if regutils.MatchIPAddr(cidr) {
		rule.IPNet = &net.IPNet{
			IP:   net.ParseIP(cidr),
			Mask: net.CIDRMask(32, 32),
		}
		return true
	}
	rule.IPNet = &net.IPNet{
		IP:   net.IPv4zero,
		Mask: net.CIDRMask(0, 32),
	}
	return false
}

func (rule *SecurityRule) IsWildMatch() bool {
	return rule.IPNet.String() == "0.0.0.0/0" &&
		rule.Protocol == PROTO_ANY &&
		len(rule.Ports) == 0 &&
		((rule.PortStart <= 0 && rule.PortEnd <= 0) || (rule.PortStart == 1 && rule.PortEnd == 65535))
}

func (rule SecurityRule) protoRelation(_rule SecurityRule) TSecurityRuleRelation {
	if rule.Direction != _rule.Direction {
		return RELATION_INDEPENDENT
	}
	if rule.Protocol == _rule.Protocol {
		if utils.IsInStringArray(rule.Protocol, []string{PROTO_ANY, PROTO_ICMP}) {
			return RELATION_IDENTICAL
		}
		if rule.PortStart <= 0 && _rule.PortStart <= 0 {
			return RELATION_IDENTICAL
		}
		if rule.PortStart > 0 && _rule.PortStart <= 0 {
			return RELATION_SUBSET
		}
		if rule.PortStart <= 0 && _rule.PortStart > 0 {
			return RELATION_SUPERSET
		}
		if rule.PortEnd+1 == _rule.PortStart {
			return RELATION_NEXT_AHEAD
		}
		if rule.PortStart == _rule.PortEnd+1 {
			return RELATION_NEXT_AFTER
		}
		if rule.PortEnd < _rule.PortStart || rule.PortStart > _rule.PortEnd {
			return RELATION_INDEPENDENT
		}
		if rule.PortStart == _rule.PortStart && rule.PortEnd == _rule.PortEnd {
			return RELATION_IDENTICAL
		}
		if rule.PortStart <= _rule.PortStart && rule.PortEnd >= _rule.PortEnd {
			return RELATION_SUPERSET
		}
		if rule.PortStart >= _rule.PortStart && rule.PortEnd <= _rule.PortEnd {
			return RELATION_SUBSET
		}
		return RELATION_OVERLAP
	}
	if rule.Protocol == PROTO_ANY {
		return RELATION_SUPERSET
	}
	if _rule.Protocol == PROTO_ANY {
		return RELATION_SUBSET
	}
	return RELATION_INDEPENDENT
}

func (rule SecurityRule) merge(r SecurityRule) SecurityRule {
	if rule.getIPKey() != r.getIPKey() {
		panic(fmt.Sprintf("rule %v ip addr not equal rule %v", rule, r))
	}
	if rule.Action != r.Action {
		panic(fmt.Sprintf("rule %v action not equal rule %v", rule, r))
	}
	rel := rule.protoRelation(r)
	switch rel {
	case RELATION_NEXT_AHEAD:
		rule.PortEnd = r.PortEnd
		return rule
	case RELATION_NEXT_AFTER:
		rule.PortStart = r.PortStart
		return rule
	case RELATION_OVERLAP:
		if rule.PortStart > r.PortStart {
			rule.PortStart = r.PortStart
		}
		if rule.PortEnd < r.PortEnd {
			rule.PortEnd = r.PortEnd
		}
		return rule
	}
	panic(fmt.Errorf("rule %s can not merge rule %s relation: %s", rule.String(), r.String(), rel))
}

func (rule SecurityRule) getIPKey() string {
	if rule.IPNet == nil || rule.IPNet.String() == "0.0.0.0/0" {
		return "0.0.0.0/0"
	}
	return rule.IPNet.String()
}

func (rule *SecurityRule) ParsePorts(seg string) error {
	if len(seg) == 0 {
		rule.Ports = []int{}
		rule.PortStart = -1
		rule.PortEnd = -1
		return nil
	} else if idx := strings.Index(seg, "-"); idx > -1 {
		segs := strings.SplitN(seg, "-", 2)
		var ps, pe int
		var err error
		if ps, err = parsePortString(segs[0]); err != nil {
			return ErrInvalidPortRange
		}
		if pe, err = parsePortString(segs[1]); err != nil {
			return ErrInvalidPortRange
		}
		if ps > pe {
			ps, pe = pe, ps
		}
		rule.PortStart = ps
		rule.PortEnd = pe
	} else if idx := strings.Index(seg, ","); idx > -1 {
		ports := make([]int, 0)
		segs := strings.Split(seg, ",")
		for _, seg := range segs {
			p, err := parsePortString(seg)
			if err != nil {
				return err
			}
			ports = append(ports, p)
		}
		rule.Ports = ports
	} else {
		p, err := parsePortString(seg)
		if err != nil {
			return err
		}
		rule.PortStart, rule.PortEnd = p, p
	}
	return nil
}

func (rule *SecurityRule) ValidateRule() error {
	if !utils.IsInStringArray(string(rule.Direction), []string{string(DIR_IN), string(DIR_OUT)}) {
		return ErrInvalidDirection
	}
	if !utils.IsInStringArray(string(rule.Action), []string{string(SecurityRuleAllow), string(SecurityRuleDeny)}) {
		return ErrInvalidAction
	}
	if !utils.IsInStringArray(rule.Protocol, []string{PROTO_ANY, PROTO_ICMP, PROTO_TCP, PROTO_UDP}) {
		return ErrInvalidProtocol
	}

	if rule.Protocol == PROTO_ICMP {
		if len(rule.Ports) > 0 || rule.PortStart > 0 || rule.PortEnd > 0 {
			return ErrInvalidProtocolICMP
		}
	}

	if rule.Protocol == PROTO_ANY {
		if len(rule.Ports) > 0 || rule.PortStart > 0 || rule.PortEnd > 0 {
			return ErrInvalidProtocolAny
		}
	}

	if len(rule.Ports) > 0 {
		for i := 0; i < len(rule.Ports); i++ {
			if rule.Ports[i] < 1 || rule.Ports[i] > 65535 {
				return ErrInvalidPort
			}
		}
	}
	if rule.PortStart > 0 || rule.PortEnd > 0 {
		if rule.PortStart < 1 {
			return ErrInvalidPortRange
		}

		if rule.PortStart > rule.PortEnd {
			return ErrInvalidPortRange
		}
		if rule.PortStart > 65535 || rule.PortEnd > 65535 {
			return ErrInvalidPortRange
		}
	}
	if rule.Priority < 1 || rule.Priority > 100 {
		return ErrInvalidPriority
	}
	return nil
}

func (rule *SecurityRule) GetPortsString() string {
	if rule.PortStart > 0 && rule.PortEnd > 0 {
		if rule.PortStart < rule.PortEnd {
			return fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd)
		}
		if rule.PortStart == rule.PortEnd {
			return fmt.Sprintf("%d", rule.PortStart)
		}
		// panic on this badness
		log.Errorf("invalid port range %d-%d", rule.PortStart, rule.PortEnd)
		return ""
	}
	if len(rule.Ports) > 0 {
		ps := []string{}
		for _, p := range rule.Ports {
			ps = append(ps, fmt.Sprintf("%d", p))
		}
		return strings.Join(ps, ",")
	}
	return ""
}

func (rule *SecurityRule) String() (result string) {
	s := []string{}
	s = append(s, string(rule.Direction)+":"+string(rule.Action))
	cidr := rule.IPNet.String()
	if cidr != "0.0.0.0/0" {
		if ones, _ := rule.IPNet.Mask.Size(); ones < 32 {
			s = append(s, cidr)
		} else {
			s = append(s, rule.IPNet.IP.String())
		}
	}

	s = append(s, rule.Protocol)
	if rule.Protocol == PROTO_TCP || rule.Protocol == PROTO_UDP {
		port := rule.GetPortsString()
		if len(port) > 0 {
			s = append(s, port)
		}
	}
	return strings.Join(s, " ")
}

func (rule *SecurityRule) equals(r *SecurityRule) bool {
	// essence of String, bom
	s0 := rule.String()
	s1 := r.String()
	return s0 == s1
}

func (rule *SecurityRule) netEquals(r *SecurityRule) bool {
	net0 := rule.IPNet.String()
	net1 := r.IPNet.String()
	return net0 == net1
}

func (rule *SecurityRule) cutOut(r *SecurityRule) SecurityRuleSet {
	srcs := securityRuleCuts{securityRuleCut{r: *rule}}
	//a := srcs
	srcs = srcs.cutOutProtocol(r.Protocol)
	srcs = srcs.cutOutIPNet(r.Protocol, r.IPNet)
	if len(r.Ports) > 0 {
		srcs = srcs.cutOutPorts(r.Protocol, []uint16(newPortsFromInts(r.Ports...)))
	} else if r.PortStart > 0 && r.PortEnd > 0 {
		srcs = srcs.cutOutPortRange(r.Protocol, uint16(r.PortStart), uint16(r.PortEnd))
	} else {
		srcs = srcs.cutOutPortsAll()
	}
	//fmt.Printf("a %s\n", a)
	//fmt.Printf("b %s\n", srcs)
	srs := srcs.securityRuleSet()
	return srs
}
