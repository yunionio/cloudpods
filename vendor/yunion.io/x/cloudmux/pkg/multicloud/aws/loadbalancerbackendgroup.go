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

package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElbBackendGroup struct {
	multicloud.SResourceBase
	AwsTags
	lb *SElb

	TargetGroupName            string   `xml:"TargetGroupName"`
	Protocol                   string   `xml:"Protocol"`
	Port                       int64    `xml:"Port"`
	VpcID                      string   `xml:"VpcId"`
	TargetType                 string   `xml:"TargetType"`
	HealthyThresholdCount      int      `xml:"HealthyThresholdCount"`
	Matcher                    Matcher  `xml:"Matcher"`
	UnhealthyThresholdCount    int      `xml:"UnhealthyThresholdCount"`
	HealthCheckPath            string   `xml:"HealthCheckPath"`
	HealthCheckProtocol        string   `xml:"HealthCheckProtocol"`
	HealthCheckPort            string   `xml:"HealthCheckPort"`
	HealthCheckIntervalSeconds int      `xml:"HealthCheckIntervalSeconds"`
	HealthCheckTimeoutSeconds  int      `xml:"HealthCheckTimeoutSeconds"`
	TargetGroupArn             string   `xml:"TargetGroupArn"`
	LoadBalancerArns           []string `xml:"LoadBalancerArns>member"`
}

func (self *SElbBackendGroup) GetILoadbalancer() cloudprovider.ICloudLoadbalancer {
	return self.lb
}

type Matcher struct {
	HTTPCode string `json:"HttpCode"`
}

func (self *SElbBackendGroup) GetId() string {
	return self.TargetGroupArn
}

func (self *SElbBackendGroup) GetName() string {
	return self.TargetGroupName
}

func (self *SElbBackendGroup) GetGlobalId() string {
	return self.GetId()
}

func (self *SElbBackendGroup) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SElbBackendGroup) Refresh() error {
	lbbg, err := self.lb.region.GetElbBackendgroup(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, lbbg)
}

func (self *SElbBackendGroup) GetSysTags() map[string]string {
	data := map[string]string{}
	data["port"] = strconv.FormatInt(self.Port, 10)
	data["target_type"] = self.TargetType
	data["health_check_protocol"] = strings.ToLower(self.HealthCheckProtocol)
	data["health_check_interval"] = strconv.Itoa(self.HealthCheckIntervalSeconds)
	return data
}

func (self *SElbBackendGroup) GetProjectId() string {
	return ""
}

func (self *SElbBackendGroup) IsDefault() bool {
	return false
}

func (self *SElbBackendGroup) GetType() string {
	return api.LB_BACKENDGROUP_TYPE_NORMAL
}

func (self *SElbBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := self.lb.region.GetELbBackends(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "GetELbBackends")
	}

	ibackends := make([]cloudprovider.ICloudLoadbalancerBackend, len(backends))
	for i := range backends {
		backends[i].group = self
		ibackends[i] = &backends[i]
	}

	return ibackends, nil
}

func (self *SElbBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
	backend, err := self.lb.region.GetELbBackend(backendId)
	if err != nil {
		return nil, errors.Wrap(err, "GetELbBackend")
	}

	backend.group = self
	return backend, nil
}

func (self *SElbBackendGroup) GetProtocolType() string {
	switch self.Protocol {
	case "TCP":
		return api.LB_LISTENER_TYPE_TCP
	case "UDP":
		return api.LB_LISTENER_TYPE_UDP
	case "HTTP":
		return api.LB_LISTENER_TYPE_HTTP
	case "HTTPS":
		return api.LB_LISTENER_TYPE_HTTPS
	case "TCP_UDP":
		return api.LB_LISTENER_TYPE_TCP_UDP
	default:
		return ""
	}
}

func (self *SElbBackendGroup) GetScheduler() string {
	return ""
}

func (self *SElbBackendGroup) GetHealthCheck() (*cloudprovider.SLoadbalancerHealthCheck, error) {
	health := &cloudprovider.SLoadbalancerHealthCheck{}
	health.HealthCheck = api.LB_BOOL_ON
	health.HealthCheckRise = self.HealthyThresholdCount
	health.HealthCheckFail = self.UnhealthyThresholdCount
	health.HealthCheckInterval = self.HealthCheckIntervalSeconds
	health.HealthCheckURI = self.HealthCheckPath
	health.HealthCheckType = strings.ToLower(self.HealthCheckProtocol)
	health.HealthCheckTimeout = self.HealthCheckTimeoutSeconds
	health.HealthCheckHttpCode = ToOnecloudHealthCode(self.Matcher.HTTPCode)
	return health, nil
}

func (self *SElbBackendGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	attrs, err := self.lb.region.GetElbBackendgroupAttributes(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "GetElbBackendgroupAttributes")
	}

	cookieTime := 0
	if t, ok := attrs["stickiness.lb_cookie.duration_seconds"]; !ok {
		cookieTime, err = strconv.Atoi(t)
	}

	ret := &cloudprovider.SLoadbalancerStickySession{
		StickySession:              attrs["stickiness.enabled"],
		StickySessionCookie:        "",
		StickySessionType:          api.LB_STICKY_SESSION_TYPE_INSERT,
		StickySessionCookieTimeout: cookieTime,
	}

	return ret, nil
}

func (self *SElbBackendGroup) AddBackendServer(serverId string, weight int, port int) (cloudprovider.ICloudLoadbalancerBackend, error) {
	backend, err := self.lb.region.AddElbBackend(self.GetId(), serverId, weight, port)
	if err != nil {
		return nil, errors.Wrap(err, "AddElbBackend")
	}

	backend.group = self
	return backend, nil
}

func (self *SElbBackendGroup) RemoveBackendServer(serverId string, weight int, port int) error {
	return self.lb.region.RemoveElbBackend(self.GetId(), serverId, weight, port)
}

func (self *SElbBackendGroup) Delete(ctx context.Context) error {
	return self.lb.region.DeleteElbBackendGroup(self.GetId())
}

func (self *SElbBackendGroup) Sync(ctx context.Context, group *cloudprovider.SLoadbalancerBackendGroup) error {
	return nil
}

func (self *SRegion) GetELbBackends(backendgroupId string) ([]SElbBackend, error) {
	params := map[string]string{
		"TargetGroupArn": backendgroupId,
	}
	ret := &SElbBackends{}
	err := self.elbRequest("DescribeTargetHealth", params, ret)
	if err != nil {
		return nil, err
	}
	return ret.TargetHealthDescriptions, nil
}

func (self *SRegion) GetELbBackend(backendId string) (*SElbBackend, error) {
	groupId, instanceId, port, err := parseElbBackendId(backendId)
	if err != nil {
		return nil, errors.Wrap(err, "parseElbBackendId")
	}
	params := map[string]string{
		"TargetGroupArn":        groupId,
		"Targets.member.1.Id":   instanceId,
		"Targets.member.1.Port": fmt.Sprintf("%d", port),
	}
	ret := &SElbBackends{}
	err = self.elbRequest("DescribeTargetHealth", params, ret)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeTargetHealth")
	}
	for i := range ret.TargetHealthDescriptions {
		if ret.TargetHealthDescriptions[i].GetGlobalId() == backendId {
			return &ret.TargetHealthDescriptions[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, backendId)
}

func parseElbBackendId(id string) (string, string, int, error) {
	segs := strings.Split(id, "::")
	if len(segs) != 3 {
		return "", "", 0, fmt.Errorf("%s is not a valid backend id", id)
	}

	port, err := strconv.Atoi(segs[2])
	if err != nil {
		return "", "", 0, fmt.Errorf("%s is not a valid backend id, %s", id, err)
	}

	return segs[0], segs[1], port, nil
}

func genElbBackendId(backendgroupId string, serverId string, port int) string {
	return strings.Join([]string{backendgroupId, serverId, strconv.Itoa(port)}, "::")
}

func (self *SRegion) AddElbBackend(backendgroupId, serverId string, weight int, port int) (*SElbBackend, error) {
	params := map[string]string{
		"TargetGroupArn":        backendgroupId,
		"Targets.member.1.Id":   serverId,
		"Targets.member.1.Port": fmt.Sprintf("%d", port),
	}
	err := self.elbRequest("RegisterTargets", params, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "RegisterTargets")
	}
	return self.GetELbBackend(genElbBackendId(backendgroupId, serverId, port))
}

func (self *SRegion) RemoveElbBackend(backendgroupId, serverId string, weight int, port int) error {
	params := map[string]string{
		"TargetGroupArn":        backendgroupId,
		"Targets.member.1.Id":   serverId,
		"Targets.member.1.Port": fmt.Sprintf("%d", port),
	}
	return self.elbRequest("DeregisterTargets", params, nil)
}

func (self *SRegion) DeleteElbBackendGroup(id string) error {
	return self.elbRequest("DeleteTargetGroup", map[string]string{"TargetGroupArn": id}, nil)
}

func (self *SRegion) RemoveElbBackends(backendgroupId string) error {
	backends, err := self.GetELbBackends(backendgroupId)
	if err != nil {
		return errors.Wrap(err, "GetELbBackends")
	}

	if len(backends) == 0 {
		return nil
	}
	for i := range backends {
		err := self.RemoveElbBackend(backendgroupId, backends[i].GetBackendId(), 0, backends[i].GetPort())
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SRegion) GetElbBackendgroupAttributes(id string) (map[string]string, error) {
	ret := struct {
		Attributes []struct {
			Key   string
			Value string
		} `xml:"Attributes>member"`
	}{}

	err := self.elbRequest("DescribeTargetGroupAttributes", map[string]string{"TargetGroupArn": id}, &ret)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, attr := range ret.Attributes {
		result[attr.Key] = attr.Value
	}
	return result, nil
}

func (self *SElbBackendGroup) GetDescription() string {
	return self.AwsTags.GetDescription()
}
