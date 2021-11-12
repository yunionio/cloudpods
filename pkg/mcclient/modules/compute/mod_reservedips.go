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

package compute

import (
	"strings"
	"sync"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ReservedIPManager struct {
	modulebase.ResourceManager
}

var (
	ReservedIPs ReservedIPManager
)

func (this *ReservedIPManager) DoBatchReleaseReservedIps(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// params format:
	// {
	//     "ips": ["1.2.3.4", "1.1.1.1"],
	// }

	ret := jsonutils.NewDict()
	_ips, e := params.GetArray("ips")
	// get ips
	if e != nil {
		return ret, e
	}
	ips := make([]string, 0)
	for _, u := range _ips {
		ip, _ := u.GetString()
		ips = append(ips, ip)
	}

	// filter ip and network pairs.
	originFilter, _ := params.Get("query")
	if originFilter == nil {
		originFilter = jsonutils.NewDict()
	}
	ipFilterOps := originFilter.(*jsonutils.JSONDict)
	arr := jsonutils.NewArray()
	filterCondition := "ip_addr.in(" + strings.Join(ips, ",") + ")"
	arr.Add(jsonutils.NewString(filterCondition))
	ipFilterOps.Add(arr, "filter")
	ipFilterOps.Add(jsonutils.NewInt(int64(1024)), "limit")
	result, err := ReservedIPs.List(s, ipFilterOps)

	if err != nil {
		return ret, err
	}

	if len(result.Data) < 1 {
		return ret, nil
	}

	// release ips within Goroutines
	var wg sync.WaitGroup
	wg.Add(len(result.Data))
	releaseIPStatus := jsonutils.NewArray()
	for _, res := range result.Data {
		networkId, _ := res.GetString("network_id")
		ip_addr, _ := res.GetString("ip_addr")
		go func(networkId, ip_addr string) {
			defer wg.Done()
			releaseOptionParams := jsonutils.NewDict()
			releaseOptionParams.Add(jsonutils.NewString(ip_addr), "ip")
			Networks.PerformAction(s, networkId, "release-reserved-ip", releaseOptionParams)
			if err != nil {
				releaseIPStatus.Add(jsonutils.NewString("[Error]" + networkId + " " + ip_addr))
			}
		}(networkId, ip_addr)
	}
	wg.Wait()
	ret.Add(releaseIPStatus, "ret")
	return ret, nil
}

func init() {
	ReservedIPs = ReservedIPManager{modules.NewComputeManager("reservedip", "reservedips",
		[]string{},
		[]string{"Id", "Network_ID", "Network", "IP_addr", "Notes", "Expired_At", "Expired", "Status"})}

	modules.RegisterCompute(&ReservedIPs)
}
