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

package ovn

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/ovsdb/schema/ovn_nb"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
)

const (
	errBadSecgroupRule = errors.Error("bad security group rule")
)

const (
	aclDirToLport   = "to-lport"
	aclDirFromLport = "from-lport"
)

func ruleToAcl(lport string, rule *agentmodels.SecurityGroupRule) (*ovn_nb.ACL, error) {
	var (
		dir    string
		action string

		match   string
		matches []string
		l3subfn string
		l4subfn string
		errs    []error
	)

	switch secrules.TSecurityRuleDirection(rule.Direction) {
	case secrules.SecurityRuleIngress:
		dir = aclDirToLport
		l3subfn = "src"
		l4subfn = "dst"
		matches = append(matches, fmt.Sprintf("outport == %q", lport))
	case secrules.SecurityRuleEgress:
		dir = aclDirFromLport
		l3subfn = "dst"
		l4subfn = "dst"
		matches = append(matches, fmt.Sprintf("inport == %q", lport))
	default:
		return nil, errors.Wrapf(errBadSecgroupRule, "unknown direction %q", rule.Direction)
	}

	switch secrules.TSecurityRuleAction(rule.Action) {
	case secrules.SecurityRuleAllow:
		action = "allow-related"
	case secrules.SecurityRuleDeny:
		action = "drop"
	default:
		return nil, errors.Wrapf(errBadSecgroupRule, "unknown action %q", rule.Action)
	}

	addL3Match := func() {
		matches = append(matches, "ip4")
		if cidr := strings.TrimSpace(rule.CIDR); cidr != "" && cidr != "0.0.0.0/0" {
			matches = append(matches, fmt.Sprintf("ip4.%s == %s", l3subfn, cidr))
		}
	}
	addL4Match := func(l4proto string) {
		var (
			l4portfn    = fmt.Sprintf("%s.%s", l4proto, l4subfn)
			portMatches []string
		)

		parsePort := func(pstr string) (int, error) {
			pn, err := strconv.ParseUint(pstr, 10, 16)
			if err != nil {
				return -1, err
			}
			return int(pn), nil
		}
		for _, pstr := range strings.Split(rule.Ports, ",") {
			pstr = strings.TrimSpace(pstr)
			if pstr == "" {
				continue
			}

			if i := strings.Index(pstr, "-"); i >= 0 {
				spstr := strings.TrimSpace(pstr[:i])
				epstr := strings.TrimSpace(pstr[i+1:])
				if spstr == "" || epstr == "" {
					continue
				}

				spn, err := parsePort(spstr)
				if err != nil {
					errs = append(errs, errors.Wrap(err, "start port"))
				}
				epn, err := parsePort(epstr)
				if err != nil {
					errs = append(errs, errors.Wrap(err, "end port"))
				}

				if spn > epn {
					spn, epn = epn, spn
				}
				if spn < epn {
					portMatches = append(portMatches,
						fmt.Sprintf("( %s >= %d && %s <= %d )",
							l4portfn, spn, l4portfn, epn))
				} else /* spn == epn */ {
					portMatches = append(portMatches,
						fmt.Sprintf("%s == %d", l4portfn, spn))
				}
			} else {
				if pn, err := parsePort(pstr); err != nil {
					errs = append(errs, errors.Wrap(err, "port"))
				} else {
					portMatches = append(portMatches,
						fmt.Sprintf("%s == %d", l4portfn, pn))
				}
			}
		}

		matches = append(matches, l4proto)
		if len(portMatches) > 0 {
			portMatch := strings.Join(portMatches, " || ")
			if len(portMatches) > 1 {
				portMatch = "( " + portMatch + " )"
			}
			matches = append(matches, portMatch)
		}
	}
	switch rule.Protocol {
	case secrules.PROTO_ANY:
		addL3Match()
	case secrules.PROTO_TCP:
		addL3Match()
		addL4Match("tcp")
	case secrules.PROTO_UDP:
		addL3Match()
		addL4Match("udp")
	case secrules.PROTO_ICMP:
		addL3Match()
		matches = append(matches, "icmp4")
	default:
		return nil, errors.Wrapf(errBadSecgroupRule, "unknown protocol %q", rule.Protocol)
	}
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}
	match = strings.Join(matches, " && ")

	acl := &ovn_nb.ACL{
		Priority:  rule.Priority,
		Direction: dir,
		Match:     match,
		Action:    action,
	}
	return acl, nil
}
