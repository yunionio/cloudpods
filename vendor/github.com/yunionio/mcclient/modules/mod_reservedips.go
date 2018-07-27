package modules

import (
	"strings"
	"sync"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
)

type ReservedIPManager struct {
	ResourceManager
}

var (
	ReservedIPs ReservedIPManager
)

func (this *ReservedIPManager) DoBatchReleaseReservedIPs(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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
	ipFilterOps := jsonutils.NewDict()
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
	ReservedIPs = ReservedIPManager{NewComputeManager("reservedip", "reservedips",
		[]string{},
		[]string{"Network_ID", "Network", "IP_addr", "Notes"})}

	registerCompute(&ReservedIPs)
}
