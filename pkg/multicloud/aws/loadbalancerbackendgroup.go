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
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/elbv2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SElbBackendGroup struct {
	region *SRegion
	lb     *SElb

	TargetGroupName            string   `json:"TargetGroupName"`
	Protocol                   string   `json:"Protocol"`
	Port                       int64    `json:"Port"`
	VpcID                      string   `json:"VpcId"`
	TargetType                 string   `json:"TargetType"`
	HealthyThresholdCount      int      `json:"HealthyThresholdCount"`
	Matcher                    Matcher  `json:"Matcher"`
	UnhealthyThresholdCount    int      `json:"UnhealthyThresholdCount"`
	HealthCheckPath            string   `json:"HealthCheckPath"`
	HealthCheckProtocol        string   `json:"HealthCheckProtocol"`
	HealthCheckPort            string   `json:"HealthCheckPort"`
	HealthCheckIntervalSeconds int      `json:"HealthCheckIntervalSeconds"`
	HealthCheckTimeoutSeconds  int      `json:"HealthCheckTimeoutSeconds"`
	TargetGroupArn             string   `json:"TargetGroupArn"`
	LoadBalancerArns           []string `json:"LoadBalancerArns"`
}

func (self *SElbBackendGroup) GetLoadbalancerId() string {
	if len(self.LoadBalancerArns) > 0 {
		return self.LoadBalancerArns[0]
	}

	return ""
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
	lbbg, err := self.region.GetElbBackendgroup(self.GetId())
	if err != nil {
		return err
	}

	err = jsonutils.Update(self, lbbg)
	if err != nil {
		return err
	}

	return nil
}

func (self *SElbBackendGroup) IsEmulated() bool {
	return false
}

func (self *SElbBackendGroup) GetMetadata() *jsonutils.JSONDict {
	metadata := jsonutils.NewDict()
	metadata.Add(jsonutils.NewInt(self.Port), "port")
	metadata.Add(jsonutils.NewString(self.TargetType), "target_type")
	metadata.Add(jsonutils.NewString(strings.ToLower(self.HealthCheckProtocol)), "health_check_protocol")
	metadata.Add(jsonutils.NewInt(int64(self.HealthCheckIntervalSeconds)), "health_check_interval")
	return metadata
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
	backends, err := self.region.GetELbBackends(self.GetId())
	if err != nil {
		return nil, err
	}

	ibackends := make([]cloudprovider.ICloudLoadbalancerBackend, len(backends))
	for i := range backends {
		backends[i].region = self.region
		backends[i].group = self
		ibackends[i] = &backends[i]
	}

	return ibackends, nil
}

func (self *SElbBackendGroup) GetILoadbalancerBackendById(backendId string) (cloudprovider.ICloudLoadbalancerBackend, error) {
	backend, err := self.region.GetELbBackend(backendId)
	if err != nil {
		return nil, err
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
	health.HealthCheckType = self.HealthCheckProtocol
	health.HealthCheckTimeout = self.HealthCheckTimeoutSeconds
	health.HealthCheckHttpCode = ToOnecloudHealthCode(self.Matcher.HTTPCode)
	return health, nil
}

func (self *SElbBackendGroup) GetStickySession() (*cloudprovider.SLoadbalancerStickySession, error) {
	attrs, err := self.region.GetElbBackendgroupAttributesById(self.GetId())
	if err != nil {
		return nil, err
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
	backend, err := self.region.AddElbBackend(self.GetId(), serverId, weight, port)
	if err != nil {
		return nil, err
	}

	backend.region = self.region
	backend.group = self
	return backend, nil
}

func (self *SElbBackendGroup) RemoveBackendServer(serverId string, weight int, port int) error {
	return self.region.RemoveElbBackend(self.GetId(), serverId, weight, port)
}

func (self *SElbBackendGroup) Delete() error {
	return self.region.DeleteElbBackendGroup(self.GetId())
}

func (self *SElbBackendGroup) Sync(group *cloudprovider.SLoadbalancerBackendGroup) error {
	return self.region.SyncELbBackendGroup(self.GetId(), group)
}

func (self *SRegion) GetELbBackends(backendgroupId string) ([]SElbBackend, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	group, err := self.GetElbBackendgroup(backendgroupId)
	if err != nil {
		return nil, err
	}

	params := &elbv2.DescribeTargetHealthInput{}
	params.SetTargetGroupArn(backendgroupId)
	output, err := client.DescribeTargetHealth(params)
	if err != nil {
		return nil, err
	}

	backends := []SElbBackend{}
	err = unmarshalAwsOutput(output, "TargetHealthDescriptions", &backends)
	if err != nil {
		return nil, err
	}

	ret := []SElbBackend{}
	for i := range backends {
		if !utils.IsInStringArray(backends[i].TargetHealth.Reason, []string{"Target.InvalidState", "Target.DeregistrationInProgress"}) {
			backends[i].region = self
			backends[i].group = group
			ret = append(ret, backends[i])
		}
	}

	return ret, nil
}

func (self *SRegion) GetELbBackend(backendId string) (*SElbBackend, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	groupId, instanceId, port, err := parseElbBackendId(backendId)
	if err != nil {
		return nil, err
	}

	params := &elbv2.DescribeTargetHealthInput{}
	desc := &elbv2.TargetDescription{}
	desc.SetPort(int64(port))
	desc.SetId(instanceId)
	params.SetTargets([]*elbv2.TargetDescription{desc})
	params.SetTargetGroupArn(groupId)
	ret, err := client.DescribeTargetHealth(params)
	if err != nil {
		return nil, err
	}

	backends := []SElbBackend{}
	err = unmarshalAwsOutput(ret, "TargetHealthDescriptions", &backends)
	if err != nil {
		return nil, err
	}

	if len(backends) == 1 {
		backends[0].region = self
		return &backends[0], nil
	}

	return nil, ErrorNotFound()
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
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	params := &elbv2.RegisterTargetsInput{}
	params.SetTargetGroupArn(backendgroupId)
	desc := &elbv2.TargetDescription{}
	desc.SetId(serverId)
	desc.SetPort(int64(port))
	params.SetTargets([]*elbv2.TargetDescription{desc})
	_, err = client.RegisterTargets(params)
	if err != nil {
		return nil, err
	}

	return self.GetELbBackend(genElbBackendId(backendgroupId, serverId, port))
}

func (self *SRegion) RemoveElbBackend(backendgroupId, serverId string, weight int, port int) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return err
	}

	params := &elbv2.DeregisterTargetsInput{}
	params.SetTargetGroupArn(backendgroupId)
	desc := &elbv2.TargetDescription{}
	desc.SetId(serverId)
	desc.SetPort(int64(port))
	params.SetTargets([]*elbv2.TargetDescription{desc})
	_, err = client.DeregisterTargets(params)
	if err != nil {
		return err
	}

	return nil
}

func (self *SRegion) DeleteElbBackendGroup(backendgroupId string) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return err
	}

	params := &elbv2.DeleteTargetGroupInput{}
	params.SetTargetGroupArn(backendgroupId)
	_, err = client.DeleteTargetGroup(params)
	if err != nil {
		return err
	}

	return nil
}

func (self *SRegion) SyncELbBackendGroup(backendgroupId string, group *cloudprovider.SLoadbalancerBackendGroup) error {
	err := self.modifyELbBackendGroup(backendgroupId, group.HealthCheck)
	if err != nil {
		return err
	}

	err = self.RemoveElbBackends(backendgroupId)
	if err != nil {
		return err
	}

	return self.AddElbBackends(backendgroupId, group.Backends)
}

func (self *SRegion) modifyELbBackendGroup(backendgroupId string, healthCheck *cloudprovider.SLoadbalancerHealthCheck) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return err
	}

	params := &elbv2.ModifyTargetGroupInput{}
	params.SetTargetGroupArn(backendgroupId)
	params.SetHealthCheckProtocol(strings.ToUpper(healthCheck.HealthCheckType))
	params.SetHealthyThresholdCount(int64(healthCheck.HealthCheckRise))

	if utils.IsInStringArray(healthCheck.HealthCheckType, []string{api.LB_HEALTH_CHECK_HTTP, api.LB_LISTENER_TYPE_HTTPS}) {
		params.SetUnhealthyThresholdCount(int64(healthCheck.HealthCheckFail))
		params.SetHealthCheckTimeoutSeconds(int64(healthCheck.HealthCheckTimeout))
		params.SetHealthCheckIntervalSeconds(int64(healthCheck.HealthCheckInterval))
		if len(healthCheck.HealthCheckURI) > 0 {
			params.SetHealthCheckPath(healthCheck.HealthCheckURI)
		}

		codes := ToAwsHealthCode(healthCheck.HealthCheckHttpCode)
		if len(codes) > 0 {
			matcher := &elbv2.Matcher{}
			matcher.SetHttpCode(codes)
			params.SetMatcher(matcher)
		}
	}

	_, err = client.ModifyTargetGroup(params)
	if err != nil {
		return err
	}

	return nil
}

func (self *SRegion) RemoveElbBackends(backendgroupId string) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return err
	}

	backends, err := self.GetELbBackends(backendgroupId)
	if err != nil {
		return err
	}

	if len(backends) == 0 {
		return nil
	}

	targets := []*elbv2.TargetDescription{}
	for i := range backends {
		target := &elbv2.TargetDescription{}
		target.SetId(backends[i].GetBackendId())
		target.SetPort(int64(backends[i].GetPort()))
		targets = append(targets, target)
	}

	params := &elbv2.DeregisterTargetsInput{}
	params.SetTargetGroupArn(backendgroupId)
	params.SetTargets(targets)
	_, err = client.DeregisterTargets(params)
	if err != nil {
		return err
	}

	return nil
}

func (self *SRegion) AddElbBackends(backendgroupId string, backends []cloudprovider.SLoadbalancerBackend) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return err
	}

	if len(backends) == 0 {
		return nil
	}

	params := &elbv2.RegisterTargetsInput{}
	params.SetTargetGroupArn(backendgroupId)
	targets := []*elbv2.TargetDescription{}
	for i := range backends {
		desc := &elbv2.TargetDescription{}
		desc.SetId(backends[i].ExternalID)
		desc.SetPort(int64(backends[i].Port))
		targets = append(targets, desc)
	}

	params.SetTargets(targets)
	_, err = client.RegisterTargets(params)
	if err != nil {
		return err
	}

	return nil
}

func (self *SRegion) GetElbBackendgroupAttributesById(backendgroupId string) (map[string]string, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, err
	}

	params := &elbv2.DescribeTargetGroupAttributesInput{}
	params.SetTargetGroupArn(backendgroupId)

	output, err := client.DescribeTargetGroupAttributes(params)
	if err != nil {
		return nil, err
	}

	attrs := []map[string]string{}
	err = unmarshalAwsOutput(output, "Attributes", &attrs)
	if err != nil {
		return nil, err
	}

	ret := map[string]string{}
	for i := range attrs {
		for k, v := range attrs[i] {
			ret[k] = v
		}
	}

	return ret, nil
}
