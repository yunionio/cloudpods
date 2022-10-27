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

package openstack

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SLoadbalancerMemberCreateParams struct {
	Name         string   `json:"name,omitempty"`
	Weight       *int     `json:"weight,omitempty"`
	AdminStateUp bool     `json:"admin_state_up,omitempty"`
	SubnetID     string   `json:"subnet_id,omitempty"`
	Address      string   `json:"address,omitempty"`
	ProtocolPort *int     `json:"protocol_port,omitempty"`
	MonitorPort  *int     `json:"monitor_port,omitempty"`
	Backup       *bool    `json:"backup,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

type SLoadbalancerMember struct {
	multicloud.SResourceBase
	OpenStackTags
	poolID             string
	region             *SRegion
	MonitorPort        int      `json:"monitor_port"`
	ProjectID          string   `json:"project_id"`
	Name               string   `json:"name"`
	Weight             int      `json:"weight"`
	Backup             bool     `json:"backup"`
	AdminStateUp       bool     `json:"admin_state_up"`
	SubnetID           string   `json:"subnet_id"`
	CreatedAt          string   `json:"created_at"`
	ProvisioningStatus string   `json:"provisioning_status"`
	MonitorAddress     string   `json:"monitor_address"`
	UpdatedAt          string   `json:"updated_at"`
	Address            string   `json:"address"`
	ProtocolPort       int      `json:"protocol_port"`
	ID                 string   `json:"id"`
	OperatingStatus    string   `json:"operating_status"`
	Tags               []string `json:"tags"`
}

func (member *SLoadbalancerMember) GetName() string {
	return member.Name
}

func (member *SLoadbalancerMember) GetId() string {
	return member.ID
}

func (member *SLoadbalancerMember) GetGlobalId() string {
	return member.GetId()
}

func (member *SLoadbalancerMember) GetStatus() string {
	switch member.ProvisioningStatus {
	case "ACTIVE":
		return api.LB_STATUS_ENABLED
	case "PENDING_CREATE":
		return api.LB_CREATING
	case "PENDING_UPDATE":
		return api.LB_SYNC_CONF
	case "PENDING_DELETE":
		return api.LB_STATUS_DELETING
	case "DELETED":
		return api.LB_STATUS_DELETED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (member *SLoadbalancerMember) IsEmulated() bool {
	return false
}

func (region *SRegion) GetLoadbalancerMenberById(poolId string, MenberId string) (*SLoadbalancerMember, error) {
	body, err := region.lbGet(fmt.Sprintf("/v2/lbaas/pools/%s/members/%s", poolId, MenberId))
	if err != nil {
		return nil, errors.Wrapf(err, "region.Get(/v2/lbaas/pools/%s/members/%s)", poolId, MenberId)
	}
	member := SLoadbalancerMember{}
	member.region = region
	member.poolID = poolId
	return &member, body.Unmarshal(&member, "member")
}

func (region *SRegion) GetLoadbalancerMenbers(poolId string) ([]SLoadbalancerMember, error) {
	members := []SLoadbalancerMember{}
	resource := fmt.Sprintf("/v2/lbaas/pools/%s/members", poolId)
	query := url.Values{}
	for {
		resp, err := region.lbList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "lbList")
		}
		part := struct {
			Members      []SLoadbalancerMember
			MembersLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		members = append(members, part.Members...)
		marker := part.MembersLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}

	for i := 0; i < len(members); i++ {
		members[i].poolID = poolId
		members[i].region = region
	}
	return members, nil
}

// serverId 转ip,对接不准确
func (region *SRegion) CreateLoadbalancerMember(poolId, serverId string, weight, port int) (*SLoadbalancerMember, error) {
	ports, err := region.GetPorts("", serverId)
	if len(ports) < 1 {
		return nil, errors.Wrap(err, "server have no port")
	}
	fixedip := SFixedIP{}
	for i := 0; i < len(ports); i++ {
		if len(ports[i].FixedIps) > 0 {
			fixedip = ports[i].FixedIps[0]
			break
		}
	}
	if len(fixedip.IpAddress) < 1 || len(fixedip.SubnetID) < 1 {
		return nil, errors.Wrap(err, "server have no fixedip")
	}
	type CreateParams struct {
		Member SLoadbalancerMemberCreateParams `json:"member"`
	}
	memberParams := CreateParams{}
	memberParams.Member.AdminStateUp = true
	memberParams.Member.Address = fixedip.IpAddress
	memberParams.Member.SubnetID = fixedip.SubnetID
	memberParams.Member.ProtocolPort = &port
	memberParams.Member.Weight = &weight

	body, err := region.lbPost(fmt.Sprintf("/v2/lbaas/pools/%s/members", poolId), jsonutils.Marshal(memberParams))
	if err != nil {
		return nil, errors.Wrapf(err, `region.lbPost(/v2/lbaas/pools/%s/members, jsonutils.Marshal(memberParams))`, poolId)
	}
	member := SLoadbalancerMember{}
	member.region = region
	member.poolID = poolId
	return &member, body.Unmarshal(&member, "member")
}

func (region *SRegion) DeleteLoadbalancerMember(poolId, memberId string) error {
	_, err := region.lbDelete(fmt.Sprintf("/v2/lbaas/pools/%s/members/%s", poolId, memberId))
	if err != nil {
		return errors.Wrapf(err, "region.lbDelete(fmt.Sprintf(/v2/lbaas/pools/%s/members/%s)", poolId, memberId)
	}
	return nil
}

func (member *SLoadbalancerMember) Refresh() error {
	newMember, err := member.region.GetLoadbalancerMenberById(member.poolID, member.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(member, newMember)
}

func (member *SLoadbalancerMember) GetWeight() int {
	return member.Weight
}

func (member *SLoadbalancerMember) GetPort() int {
	return member.ProtocolPort
}

func (member *SLoadbalancerMember) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (member *SLoadbalancerMember) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

// 网络地址映射设备
func (member *SLoadbalancerMember) GetBackendId() string {
	ports, err := member.region.GetPorts("", "")
	if err != nil {
		log.Errorln(errors.Wrap(err, "member.region.GetPorts()"))
	}
	for i := 0; i < len(ports); i++ {
		for j := 0; j < len(ports[i].FixedIps); j++ {
			fixedIP := ports[i].FixedIps[j]
			if fixedIP.SubnetID == member.SubnetID && fixedIP.IpAddress == member.Address {
				return ports[i].DeviceID
			}
		}
	}
	return ""
}

func (member *SLoadbalancerMember) GetIpAddress() string {
	return ""
}

func (member *SLoadbalancerMember) GetProjectId() string {
	return member.ProjectID
}

func (region *SRegion) UpdateLoadBalancerMemberWtight(poolId, memberId string, weight int) error {
	params := jsonutils.NewDict()
	poolParam := jsonutils.NewDict()
	poolParam.Add(jsonutils.NewInt(int64(weight)), "weight")
	params.Add(poolParam, "member")
	_, err := region.lbUpdate(fmt.Sprintf("/v2/lbaas/pools/%s/members/%s", poolId, memberId), params)
	if err != nil {
		return errors.Wrapf(err, "region.lbUpdate(fmt.Sprintf(/v2/lbaas/pools/%s/members/%s", poolId, memberId)
	}
	return nil
}

func (member *SLoadbalancerMember) SyncConf(ctx context.Context, port, weight int) error {
	if port > 0 {
		log.Warningf("Elb backend SyncConf unsupport modify port")
	}
	// ensure member status
	err := waitLbResStatus(member, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waitLbResStatus(member, 10*time.Second, 8*time.Minute)")
	}
	err = member.region.UpdateLoadBalancerMemberWtight(member.poolID, member.ID, weight)
	if err != nil {
		return errors.Wrapf(err, "member.region.UpdateLoadBalancerMemberWtight(%s,%s,%d)", member.poolID, member.ID, weight)
	}
	err = waitLbResStatus(member, 10*time.Second, 8*time.Minute)
	if err != nil {
		return errors.Wrap(err, "waitLbResStatus(member, 10*time.Second, 8*time.Minute)")
	}
	return nil
}
