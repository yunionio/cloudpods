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

package misc

import (
	"context"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/influxdb"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

func PingProbe(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	err := func() error {
		if options.Options.DisablePingProbe {
			return nil
		}
		isRoot := sysutils.IsRootPermission()
		if !isRoot {
			return errors.Error("require root permissions")
		}
		s := auth.GetAdminSession(ctx, options.Options.Region)
		networks := []api.NetworkDetails{}
		for {
			params := map[string]interface{}{
				"offset":     len(networks),
				"limit":      "10",
				"cloud_env":  api.CLOUD_ENV_ON_PREMISE,
				"scope":      rbacscope.ScopeSystem,
				"is_classic": true,
			}
			resp, err := compute.Networks.List(s, jsonutils.Marshal(params))
			if err != nil {
				return errors.Wrapf(err, "Networks.List")
			}
			part := []api.NetworkDetails{}
			err = jsonutils.Update(&part, resp.Data)
			if err != nil {
				return errors.Wrapf(err, "jsonutils.Update")
			}
			networks = append(networks, part...)
			if len(networks) >= resp.Total {
				break
			}
		}
		metrics := make([]influxdb.SMetricData, 0)
		for i := range networks {
			network := sNetwork{networks[i]}
			m, err := pingProbeNetwork(s, network)
			if err != nil {
				log.Errorf("pingProbeNetwork")
				continue
			}
			metrics = append(metrics, m...)
		}
		urls, err := tsdb.GetDefaultServiceSourceURLs(s, options.Options.SessionEndpointType)
		if err != nil {
			return errors.Wrap(err, "GetServiceURLs")
		}
		return influxdb.SendMetrics(urls, options.Options.InfluxDatabase, metrics, false)
	}()
	if err != nil {
		log.Errorf("PingProb error: %v", err)
	}
}

type sNetwork struct {
	api.NetworkDetails
}

func getNetworkAddrMap(s *mcclient.ClientSession, netId string) (map[string]api.SNetworkUsedAddress, error) {
	addrListJson, err := compute.Networks.GetSpecific(s, netId, "addresses", nil)
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

func pingProbeNetwork(s *mcclient.ClientSession, net sNetwork) ([]influxdb.SMetricData, error) {
	metrics := make([]influxdb.SMetricData, 0)
	if net.CloudEnv != api.CLOUD_ENV_ON_PREMISE {
		return nil, errors.Wrap(errors.ErrInvalidStatus, "not onpremise network")
	}
	if net.GuestGateway == "" {
		return nil, errors.Wrap(errors.ErrInvalidStatus, "unreachable network, empty gateway")
	}
	addrStart, err := netutils.NewIPV4Addr(net.GuestIpStart)
	if err != nil {
		return nil, errors.Wrapf(err, "NewIPV4Addr %s", net.GuestIpStart)
	}
	addrEnd, err := netutils.NewIPV4Addr(net.GuestIpEnd)
	if err != nil {
		return nil, errors.Wrapf(err, "NewIPV4Addr %s", net.GuestIpEnd)
	}
	log.Infof("ping address %s - %s", addrStart, addrEnd)
	pingAddrs := make([]string, 0)
	for addr := addrStart; addr <= addrEnd; addr = addr.StepUp() {
		addrStr := addr.String()
		pingAddrs = append(pingAddrs, addrStr)
	}
	pingResults, err := Ping(pingAddrs,
		options.Options.PingProbeOptions.ProbeCount,
		options.Options.PingProbeOptions.TimeoutSecond,
		options.Options.PingProbeOptions.Debug,
	)
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
				_, err := compute.ReservedIPs.Update(s, netAddr.OwnerId, params)
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
				_, err = compute.Networks.PerformAction(s, net.Id, "reserve-ip", params)
				if err != nil {
					log.Errorf("failed to reserve ip %s: %s", addrStr, err)
				}
			}
		}
		log.Debugf("%s %s allocated %v", addrStr, netAddr, allocated)
	}
	return metrics, nil
}
