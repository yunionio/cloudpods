package secrules

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"yunion.io/x/pkg/util/regutils"
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

type SecurityRule struct {
	Priority  int // [1, 100]
	Action    TSecurityRuleAction
	IPNet     *net.IPNet
	Protocol  string
	Direction TSecurityRuleDirection
	PortStart int
	PortEnd   int
	Ports     []int
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

var (
	ErrInvalidDirection = errors.New("invalid direction")
	ErrInvalidAction    = errors.New("invalid action")
	ErrInvalidNet       = errors.New("invalid net")
	ErrInvalidIPAddr    = errors.New("invalid ip address")
	ErrInvalidProtocol  = errors.New("invalid protocol")
	ErrInvalidPortRange = errors.New("invalid port range")
	ErrInvalidPort      = errors.New("invalid port")
)

func parsePortString(ps string) (int, error) {
	p, err := strconv.ParseUint(ps, 10, 16)
	if err != nil || p == 0 {
		return 0, ErrInvalidPort
	}
	return int(p), nil
}

func ParseSecurityRule(pattern string) (*SecurityRule, error) {
	rule := &SecurityRule{}
	for _, direction := range []TSecurityRuleDirection{SecurityRuleIngress, SecurityRuleEgress} {
		if pattern[:len(direction)+1] == string(direction)+":" {
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
			// NOTE regutils.MatchCIDR actually also matches IP address without prefix length
			if regutils.MatchCIDR(seg) {
				if idx := strings.Index(seg, "/"); idx > -1 {
					if _, ipnet, err := net.ParseCIDR(seg); err != nil {
						return nil, ErrInvalidNet
					} else {
						rule.IPNet = ipnet
					}
				} else if ip := net.ParseIP(seg); ip != nil {
					rule.IPNet = &net.IPNet{
						IP:   ip,
						Mask: net.CIDRMask(32, 32),
					}
				} else {
					return nil, ErrInvalidIPAddr
				}
			} else {
				rule.IPNet = &net.IPNet{
					IP:   net.IPv4zero,
					Mask: net.CIDRMask(0, 32),
				}
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
			if len(seg) == 0 {
				status = SEG_END
			} else if idx := strings.Index(seg, "-"); idx > -1 {
				segs := strings.SplitN(seg, "-", 2)
				var ps, pe int
				var err error
				if ps, err = parsePortString(segs[0]); err != nil {
					return nil, ErrInvalidPortRange
				}
				if pe, err = parsePortString(segs[1]); err != nil {
					return nil, ErrInvalidPortRange
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
						return nil, err
					}
					ports = append(ports, p)
				}
				rule.Ports = ports
			} else {
				p, err := parsePortString(seg)
				if err != nil {
					return nil, err
				}
				rule.PortStart, rule.PortEnd = p, p
			}
			status = SEG_END
		}
	}
	return rule, nil
}

func (rule *SecurityRule) IsWildMatch() bool {
	return rule.IPNet.String() == "0.0.0.0/0" &&
		rule.Protocol == PROTO_ANY &&
		len(rule.Ports) == 0 &&
		rule.PortStart == 0 &&
		rule.PortEnd == 0
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
		if rule.PortStart > 0 && rule.PortEnd > 0 {
			if rule.PortStart < rule.PortEnd {
				s = append(s, fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd))
			} else if rule.PortStart == rule.PortEnd {
				s = append(s, fmt.Sprintf("%d", rule.PortStart))
			} else {
				// panic on this badness
			}
		} else if len(rule.Ports) > 0 {
			ps := []string{}
			for _, p := range rule.Ports {
				ps = append(ps, fmt.Sprintf("%d", p))
			}
			s = append(s, strings.Join(ps, ","))
		}
	}
	return strings.Join(s, " ")
}
