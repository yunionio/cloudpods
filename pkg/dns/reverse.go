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
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/etcd/msg"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/coredns/coredns/request"

	"yunion.io/x/onecloud/pkg/compute/models"
)

// Reverse implements the ServiceBackend interface
func (r *SRegionDNS) Reverse(state request.Request, exact bool, opt plugin.Options) (services []msg.Service, err error) {
	ip := dnsutil.ExtractAddressFromReverse(state.Name())
	if ip == "" {
		_, e := r.Records(state, exact)
		return nil, e
	}
	records, err := r.getNameForIp(ip, state)
	return records, err
}

func (r *SRegionDNS) getNameForIp(ip string, state request.Request) ([]msg.Service, error) {
	req, e := parseRequest(state)
	if e != nil {
		return nil, e
	}

	// 1. try local dns records table
	records := models.DnsRecordManager.QueryDnsIps(req.ProjectId(), req.Name(), req.Type())
	for _, rec := range records {
		return []msg.Service{{Host: rec.Addr, TTL: uint32(rec.Ttl)}}, nil
	}

	// 2. try hosts table
	host := models.HostnetworkManager.GetHostByAddress(ip)
	if host != nil {
		return []msg.Service{{Host: r.joinDomain(host.Name), TTL: defaultTTL}}, nil
	}

	// 3. try guests table
	guest := models.GuestnetworkManager.GetGuestByAddress(ip)
	if guest != nil {
		return []msg.Service{{Host: r.joinDomain(guest.Name), TTL: defaultTTL}}, nil
	}
	return nil, errNotFound
}

func (r *SRegionDNS) joinDomain(name string) string {
	return strings.Join([]string{name, r.PrimaryZone}, ".")
}
