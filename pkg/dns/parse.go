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

	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"

	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/compute/models"
)

type recordRequest struct {
	state request.Request
	// domainSegs   []string
	srcProjectId string
	srcInCloud   bool
	// network      *models.SNetwork
}

func parseRequest(state request.Request) *recordRequest {
	// base, _ := dnsutil.TrimZone(state.Name(), state.Zone)
	// segs := dns.SplitDomainName(base)
	r := &recordRequest{
		state: state,
		// domainSegs: segs,
	}
	srcIP := r.SrcIP4()
	// NOTE the check on networks_tbl is a hack, we should be more specific
	// by querying only guest networks, host networks, and others like
	// loadbalancer network the to come.
	//
	// Order matters here, we want to find the srcIP project as accurately
	// as possible

	if guest := models.GuestnetworkManager.GetGuestByAddress(srcIP, ""); guest != nil {
		r.srcProjectId = guest.ProjectId
		r.srcInCloud = true
	} else if _, err := models.NetworkManager.GetOnPremiseNetworkOfIP(srcIP, "", tristate.None); err == nil {
		// r.srcProjectId = "" // no specific project
		r.srcInCloud = true
	}
	return r
}

func (r recordRequest) Name() string {
	//fullName, _ := dnsutil.TrimZone(r.state.Name(), "")
	name := r.state.Name()
	name = strings.TrimSuffix(name, ".")
	return name
}

func (r recordRequest) Zone() string {
	return r.state.Zone
}

func (r recordRequest) IsPlainName() bool {
	nl := dns.CountLabel(r.Name())
	return nl == 1
}

func (r recordRequest) QueryName() string {
	seps := strings.Split(r.Name(), ".")
	if len(seps) == 0 {
		return ""
	}
	return seps[0]
}

func (r recordRequest) Type() string {
	return DNSTypeMap[r.state.QType()]
}

func (r recordRequest) IsSRV() bool {
	return r.Type() == DNSTypeMap[dns.TypeSRV]
}

func (r recordRequest) SrcIP4() string {
	ip := r.state.IP()
	return ip
}

func (r recordRequest) ProjectId() string {
	return r.srcProjectId
}

func (r recordRequest) SrcInCloud() bool {
	return r.srcInCloud
}

/*type K8sQueryInfo struct {
	ServiceName string
	Namespace   string
}

func (r recordRequest) GetK8sQueryInfo() K8sQueryInfo {
	parts := strings.SplitN(r.Name(), ".", 3)
	var svcName string
	var namespace string
	if len(parts) >= 2 {
		svcName = parts[0]
		namespace = parts[1]
	} else {
		svcName = parts[0]
		namespace = "default"
	}
	return K8sQueryInfo{
		ServiceName: svcName,
		Namespace:   namespace,
	}
}*/
