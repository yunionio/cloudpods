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

package utils

import (
	"context"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/apis"
	ansible_api "yunion.io/x/onecloud/pkg/apis/ansible"
	comapi "yunion.io/x/onecloud/pkg/apis/compute"
)

type sProxyEndpoint struct {
	Id      string
	Address string
}

var ErrCannotReachInfluxbd = errors.Error("no suitable network to reach influxdb")

func GetLocalArgs(serverDetails *comapi.ServerDetails, influxdbUrl string) map[string]interface{} {
	info := sServerInfo{}
	info.serverDetails = serverDetails
	info.ServerId = serverDetails.Id
	networkIds := sets.NewString()
	for _, nic := range serverDetails.Nics {
		networkIds.Insert(nic.NetworkId)
		info.VpcId = nic.VpcId
	}
	info.NetworkIds = networkIds.UnsortedList()

	return getArgs(&info, influxdbUrl)
}

func GetArgs(ctx context.Context, serverId, proxyEndpointId string, others interface{}) (map[string]interface{}, error) {
	host, ok := others.(*ansible_api.AnsibleHost)
	if !ok {
		return nil, errors.Error("unknown others, want *AnsibleHost")
	}
	info, err := GetServerInfo(ctx, serverId)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get serverInfo of server %s", serverId)
	}
	monitorUrl := info.serverDetails.MonitorUrl
	log.Infof("TSDB monitor Url: %s", monitorUrl)
	foundSvc := false
	errs := []error{}
	for _, svcName := range []string{apis.SERVICE_TYPE_INFLUXDB, apis.SERVICE_TYPE_VICTORIA_METRICS} {
		if tsdbUrl, err := FindValidServiceUrl(ctx, Service{svcName, monitorUrl}, proxyEndpointId, info, host); err != nil {
			errs = append(errs, errors.Wrapf(err, "unable to convertInfluxdbUrl %s", monitorUrl))
		} else {
			monitorUrl = tsdbUrl
			foundSvc = true
			break
		}
	}
	if !foundSvc {
		return nil, errors.Wrapf(errors.NewAggregate(errs), "convert TSDB service URL")
	}
	if len(monitorUrl) == 0 {
		return nil, errors.Wrap(ErrCannotReachInfluxbd, "please create usable Proxy Endpoint for server and try again")
	}
	return getArgs(&info, monitorUrl), nil
}
func getArgs(info *sServerInfo, influxdbUrl string) map[string]interface{} {
	tags := map[string]string{
		"host":             info.serverDetails.Host,
		"host_id":          info.serverDetails.HostId,
		"vm_id":            info.serverDetails.Id,
		"vm_ip":            info.serverDetails.IPs,
		"vm_name":          info.serverDetails.Name,
		"zone":             info.serverDetails.Zone,
		"zone_id":          info.serverDetails.ZoneId,
		"zone_ext_id":      info.serverDetails.ZoneExtId,
		"os_type":          info.serverDetails.OsType,
		"status":           info.serverDetails.Status,
		"cloudregion":      info.serverDetails.Cloudregion,
		"cloudregion_id":   info.serverDetails.CloudregionId,
		"region_ext_id":    info.serverDetails.RegionExtId,
		"tenant":           info.serverDetails.Project,
		"tenant_id":        info.serverDetails.ProjectId,
		"brand":            info.serverDetails.Brand,
		"scaling_group_id": info.serverDetails.ScalingGroupId,
		"domain_id":        info.serverDetails.DomainId,
		"project_domain":   info.serverDetails.ProjectDomain,
	}
	telegrafTags := make([]map[string]string, 0, len(tags))
	for name, value := range tags {
		telegrafTags = append(telegrafTags, map[string]string{
			"tag_name":  name,
			"tag_value": value,
		})
	}
	ret := map[string]interface{}{
		"influxdb_url":         influxdbUrl,
		"influxdb_name":        "telegraf",
		"telegraf_global_tags": telegrafTags,
	}
	if info.serverDetails.Hypervisor == comapi.HYPERVISOR_BAREMETAL {
		ret["server_type"] = "baremetal"
	}
	return ret
}
