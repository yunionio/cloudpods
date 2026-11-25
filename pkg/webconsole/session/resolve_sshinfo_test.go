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

package session

import (
	"testing"

	"yunion.io/x/jsonutils"

	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

func TestPickHostByIp(t *testing.T) {
	newHosts := func(hosts ...compute_api.HostDetails) []jsonutils.JSONObject {
		ret := make([]jsonutils.JSONObject, len(hosts))
		for i := range hosts {
			ret[i] = jsonutils.Marshal(hosts[i])
		}
		return ret
	}

	// the list api matches a single ip as an unanchored regexp, so the
	// result may contain hosts whose address merely starts with the
	// requested one
	hosts := newHosts(
		compute_api.HostDetails{SHost: compute_api.SHost{AccessIp: "10.22.42.26"}},
		compute_api.HostDetails{SHost: compute_api.SHost{AccessIp: "10.22.42.2"}},
	)
	host := pickHostByIp(hosts, "10.22.42.2")
	if host == nil {
		t.Fatalf("expected host 10.22.42.2 to be found")
	}
	if host.AccessIp != "10.22.42.2" {
		t.Fatalf("picked host %q, want %q", host.AccessIp, "10.22.42.2")
	}

	// a fuzzy match only (10.22.42.26 for 10.22.42.2) must not be accepted,
	// otherwise the ssh session is redirected to another host
	if host := pickHostByIp(hosts[:1], "10.22.42.2"); host != nil {
		t.Fatalf("host %q must not match ip %q", host.AccessIp, "10.22.42.2")
	}

	// the ip may be one of the host's management/nic addresses
	byIpmi := newHosts(compute_api.HostDetails{
		SHost: compute_api.SHost{AccessIp: "192.168.1.10", IpmiIp: "10.22.42.2"},
	})
	if host := pickHostByIp(byIpmi, "10.22.42.2"); host == nil || host.AccessIp != "192.168.1.10" {
		t.Fatalf("expected host matched by ipmi ip, got %v", host)
	}
	byNic := newHosts(compute_api.HostDetails{
		SHost:   compute_api.SHost{AccessIp: "192.168.1.10"},
		NicInfo: []*types.SNic{{IpAddr: "10.22.42.2"}},
	})
	if host := pickHostByIp(byNic, "10.22.42.2"); host == nil || host.AccessIp != "192.168.1.10" {
		t.Fatalf("expected host matched by nic ip, got %v", host)
	}
}

func TestOrderServerCandidatesByIp(t *testing.T) {
	newServers := func(servers ...compute_api.ServerDetails) []jsonutils.JSONObject {
		ret := make([]jsonutils.JSONObject, len(servers))
		for i := range servers {
			ret[i] = jsonutils.Marshal(servers[i])
		}
		return ret
	}

	servers := newServers(
		compute_api.ServerDetails{IPs: "10.22.42.26"},
		compute_api.ServerDetails{IPs: "10.22.42.2"},
	)
	got := orderServerCandidatesByIp(servers, "10.22.42.2")
	if len(got) != 1 || got[0].IPs != "10.22.42.2" {
		t.Fatalf("expected only the server with ip 10.22.42.2, got %v", got)
	}

	// without an exact match, keep all candidates so the caller can still
	// validate them through the server networks
	got = orderServerCandidatesByIp(servers[:1], "10.22.42.2")
	if len(got) != 1 || got[0].IPs != "10.22.42.26" {
		t.Fatalf("expected fallback candidates, got %v", got)
	}

	// eip and nic addresses are also valid matches
	if !serverHasIp(&compute_api.ServerDetails{Eip: "10.22.42.2"}, "10.22.42.2") {
		t.Fatalf("eip should match")
	}
	if !serverHasIp(&compute_api.ServerDetails{Nics: []compute_api.GuestnetworkShortDesc{{IpAddr: "10.22.42.2"}}}, "10.22.42.2") {
		t.Fatalf("nic ip should match")
	}
	if serverHasIp(&compute_api.ServerDetails{IPs: "10.22.42.26"}, "10.22.42.2") {
		t.Fatalf("10.22.42.26 must not match 10.22.42.2")
	}
}
