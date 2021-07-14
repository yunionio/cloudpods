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

package collectors

import (
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/influxdb"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

func pingProbeCoolector(s *mcclient.ClientSession, args *common.ReportOptions) error {
	isRoot := sysutils.IsRootPermission()
	if !isRoot {
		return errors.Error("require root permissions")
	}
	metrics := make([]influxdb.SMetricData, 0)
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(api.CLOUD_ENV_ON_PREMISE), "cloud_env")
	params.Add(jsonutils.NewString(string(rbacutils.ScopeSystem)), "scope")
	params.Add(jsonutils.JSONTrue, "is_classic")
	listAll(s, modules.Networks.List, params, func(data jsonutils.JSONObject) error {
		m, err := pingProbeNetwork(s, data, &args.PingProbeOptions)
		if err != nil {
			return err
		}
		if len(m) > 0 {
			metrics = append(metrics, m...)
		}
		return nil
	})
	return sendMetrics(s, metrics, args.Debug)
}

type sNetwork struct {
	Id           string
	Name         string
	GuestGateway string
	GuestIpStart string
	GuestIpEnd   string
	Region       string
	RegionId     string
	ServerType   string
	Vpc          string
	VpcId        string
	Wire         string
	WireId       string
	Zone         string
	ZoneId       string
	CloudEnv     string
}

func getNetworkAddrMap(s *mcclient.ClientSession, netId string) (map[string]api.SNetworkUsedAddress, error) {
	addrListJson, err := modules.Networks.GetSpecific(s, netId, "addresses", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetSpecific addresses")
	}
	addrList := make([]api.SNetworkUsedAddress, 0)
	err = addrListJson.Unmarshal(&addrList, "addresses")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal addreses")
	}
	addrMap := make(map[string]api.SNetworkUsedAddress)
	for i := range addrList {
		addrMap[addrList[i].IpAddr] = addrList[i]
	}
	return addrMap, nil
}

func pingProbeNetwork(s *mcclient.ClientSession, data jsonutils.JSONObject,
	args *common.PingProbeOptions) ([]influxdb.SMetricData, error) {
	metrics := make([]influxdb.SMetricData, 0)
	net := &sNetwork{}
	err := data.Unmarshal(&net)
	if err != nil {
		log.Errorf("Unmarshal network %s: %s", data, err)
		return nil, errors.Wrap(err, "Unmarshal network")
	}
	if net.CloudEnv != api.CLOUD_ENV_ON_PREMISE {
		log.Errorf("not an onpremise network: %s", data)
		return nil, errors.Wrap(errors.ErrInvalidStatus, "not onpremise network")
	}
	if net.GuestGateway == "" {
		log.Errorf("no valid gateway %s", data)
		return nil, errors.Wrap(errors.ErrInvalidStatus, "unreachable network, empty gateway")
	}
	addrStart, err := netutils.NewIPV4Addr(net.GuestIpStart)
	if err != nil {
		log.Errorf("unmarshal start address %s: %s", net.GuestIpStart, err)
		return nil, errors.Wrapf(err, "NewIPV4Addr %s", net.GuestIpStart)
	}
	addrEnd, err := netutils.NewIPV4Addr(net.GuestIpEnd)
	if err != nil {
		log.Errorf("unmarshal end address %s: %s", net.GuestIpEnd, err)
		return nil, errors.Wrapf(err, "NewIPV4Addr %s", net.GuestIpEnd)
	}
	log.Infof("ping address %s - %s", addrStart, addrEnd)
	pingAddrs := make([]string, 0)
	for addr := addrStart; addr <= addrEnd; addr = addr.StepUp() {
		addrStr := addr.String()
		pingAddrs = append(pingAddrs, addrStr)
	}
	pingResults, err := Ping(pingAddrs, args.ProbeCount, time.Second*time.Duration(args.TimeoutSecond), args.Debug)
	if err != nil {
		return nil, errors.Wrap(err, "Ping")
	}

	addrMap, err := getNetworkAddrMap(s, net.Id)
	if err != nil {
		return nil, errors.Wrap(err, "getNetworkAddrMap")
	}

	now := time.Now().UTC()
	for addr := addrStart; addr <= addrEnd; addr = addr.StepUp() {
		addrStr := addr.String()
		pingResult := pingResults[addrStr]
		netAddr, allocated := addrMap[addrStr]
		if allocated {
			if netAddr.OwnerType == api.RESERVEDIP_RESOURCE_TYPES {
				loss := pingResult.Loss()
				status := api.RESERVEDIP_STATUS_OFFLINE
				if loss < 100 {
					status = api.RESERVEDIP_STATUS_ONLINE
				}
				params := jsonutils.NewDict()
				params.Add(jsonutils.NewString(status), "status")
				_, err := modules.ReservedIPs.Update(s, netAddr.OwnerId, params)
				if err != nil {
					log.Errorf("update reserved ip %s status fail: %s", addrStr, err)
				}
			} else {
				// send metrics
				metric := influxdb.SMetricData{}
				metric.Name = "ping"
				metric.Timestamp = now
				metric.Tags = []influxdb.SKeyValue{
					{
						Key:   "ip_addr",
						Value: addrStr,
					},
					{
						Key:   "owner_type",
						Value: netAddr.OwnerType,
					},
					{
						Key:   "owner_id",
						Value: netAddr.OwnerId,
					},
					{
						Key:   "owner",
						Value: netAddr.Owner,
					},
				}
				loss := pingResult.Loss()
				max, avg, min := pingResult.Rtt()
				metric.Metrics = []influxdb.SKeyValue{
					{
						Key:   "loss",
						Value: strconv.FormatInt(int64(loss), 10),
					},
					{
						Key:   "rtt_ms_avg",
						Value: strconv.FormatInt(int64(avg/time.Millisecond), 10),
					},
					{
						Key:   "rtt_ms_max",
						Value: strconv.FormatInt(int64(max/time.Millisecond), 10),
					},
					{
						Key:   "rtt_ms_min",
						Value: strconv.FormatInt(int64(min/time.Millisecond), 10),
					},
				}
				metrics = append(metrics, metric)
			}
		} else {
			loss := pingResult.Loss()
			if loss < 100 {
				// reserve ip
				log.Debugf("Free address %s is responding ping, reserve the address", addrStr)
				params := jsonutils.NewDict()
				params.Add(jsonutils.NewStringArray([]string{addrStr}), "ips")
				params.Add(jsonutils.NewString("ping detect online free IP"), "notes")
				params.Add(jsonutils.NewString(api.RESERVEDIP_STATUS_ONLINE), "status")
				_, err := modules.Networks.PerformAction(s, net.Id, "reserve-ip", params)
				if err != nil {
					log.Errorf("failed to reserve ip %s: %s", addrStr, err)
				}
			}
		}
		log.Debugf("%s %s allocated %v", addrStr, netAddr, allocated)
	}
	return metrics, nil
}

func init() {
	shellutils.R(&common.ReportOptions{}, "ping-probe", "Ping probe IPv4 address", pingProbeCoolector)
}
