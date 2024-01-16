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
	req := parseRequest(state)

	// 1. try local dns records table
	recs, _ := models.DnsRecordManager.QueryPtr(req.ProjectId(), ip)
	if len(recs) > 0 {
		services := make([]msg.Service, 0, len(recs))
		for _, rec := range recs {
			services = append(services, msg.Service{Host: rec.DnsName, TTL: uint32(rec.TTL)})
		}
		return services, nil
	}

	// 2. try guests table
	guest := models.GuestnetworkManager.GetGuestByAddress(ip, req.ProjectId())
	if guest != nil {
		return []msg.Service{{Host: r.joinDomain(guest.Hostname), TTL: defaultTTL}}, nil
	}

	// 3. try hosts table
	host := models.HostnetworkManager.GetHostByAddress(ip)
	if host != nil {
		return []msg.Service{{Host: r.joinDomain(host.Name), TTL: defaultTTL}}, nil
	}

	return nil, errNotFound
}

func (r *SRegionDNS) joinDomain(name string) string {
	return strings.Join([]string{name, r.PrimaryZone}, ".")
}
