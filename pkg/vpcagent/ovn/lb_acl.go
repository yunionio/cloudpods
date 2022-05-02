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
	"sort"
	"strings"

	"yunion.io/x/ovsdb/schema/ovn_nb"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
)

const (
	errBadLblistenerAcl = errors.Error("bad loadbalancer listener acl rule")
)

func lblistenerToAcls(lport string, lblistener *agentmodels.LoadbalancerListener) ([]*ovn_nb.ACL, error) {
	if aclStatus := lblistener.AclStatus; aclStatus != computeapi.LB_BOOL_ON {
		return nil, nil
	}

	var cidrs []string
	lbacl := lblistener.LoadbalancerAcl
	if lbacl != nil || lbacl.AclEntries != nil {
		aclEntries := *lbacl.AclEntries
		for _, aclEntry := range aclEntries {
			if aclEntry != nil && aclEntry.Cidr != "" {
				cidrs = append(cidrs, aclEntry.Cidr)
			}
		}
	}

	var (
		aclType       = lblistener.AclType
		action        string
		actionDefault string
	)
	switch aclType {
	case computeapi.LB_ACL_TYPE_BLACK:
		if len(cidrs) == 0 {
			// nothing to black out and it will be allowed by
			// default by ovn for no matching
			return nil, nil
		}
		action = "drop"
		actionDefault = "allow-related"
	case computeapi.LB_ACL_TYPE_WHITE:
		action = "allow-related"
		actionDefault = "drop"
	default:
		return nil, errors.Wrapf(errBadLblistenerAcl, "unknown acl type %q", aclType)
	}

	var portmatch = fmt.Sprintf("outport == %q", lport)

	var l3match string
	if len(cidrs) > 0 {
		var l3matches = make([]string, len(cidrs))
		sort.Strings(cidrs)
		for i, cidr := range cidrs {
			l3matches[i] = fmt.Sprintf("ip4.src == %s", cidr)
		}
		l3match = "(" + strings.Join(l3matches, " || ") + ")"
	}

	var (
		l4proto string
		l4port  int
		l4match string
	)
	switch listenerType := lblistener.ListenerType; listenerType {
	case computeapi.LB_LISTENER_TYPE_TCP,
		computeapi.LB_LISTENER_TYPE_HTTP,
		computeapi.LB_LISTENER_TYPE_HTTPS:
		l4proto = "tcp"
	case computeapi.LB_LISTENER_TYPE_UDP:
		l4proto = "udp"
	default:
		return nil, errors.Wrapf(errBadLblistenerAcl, "unknown listener type %q", listenerType)
	}
	l4port = lblistener.ListenerPort
	l4match = fmt.Sprintf("%s.dst == %d", l4proto, l4port)

	const (
		prio200 = 200
		prio100 = 100
	)
	var (
		match200 string
		match100 string
	)
	match200 = portmatch
	if l3match != "" {
		match200 += " && " + l3match
	}
	match200 += " && " + l4match
	match100 = portmatch + " && " + l4match

	acls := []*ovn_nb.ACL{
		&ovn_nb.ACL{
			Priority:  200,
			Direction: aclDirToLport,
			Match:     match200,
			Action:    action,
		},
		&ovn_nb.ACL{
			Priority:  100,
			Direction: aclDirToLport,
			Match:     match100,
			Action:    actionDefault,
		},
	}

	return acls, nil
}
