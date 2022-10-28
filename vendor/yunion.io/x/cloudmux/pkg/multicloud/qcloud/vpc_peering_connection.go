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

package qcloud

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SVpcPC struct {
	multicloud.SResourceBase
	QcloudTags
	vpc                   *SVpc
	VpcID                 string `json:"vpcId"`
	UnVpcID               string `json:"unVpcId"`
	PeerVpcID             string `json:"peerVpcId"`
	UnPeerVpcID           string `json:"unPeerVpcId"`
	AppID                 string `json:"appId"`
	PeeringConnectionID   string `json:"peeringConnectionId"`
	PeeringConnectionName string `json:"peeringConnectionName"`
	State                 int    `json:"state"`
	CreateTime            string `json:"createTime"`
	Uin                   string `json:"uin"`
	PeerUin               string `json:"peerUin"`
	Region                string `json:"region"`
	PeerRegion            string `json:"peerRegion"`
}

func (region *SRegion) DescribeVpcPeeringConnections(vpcId string, peeringConnectionId string, offset int, limit int) ([]SVpcPC, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["offset"] = fmt.Sprintf("%d", offset)
	params["limit"] = fmt.Sprintf("%d", limit)
	if len(vpcId) > 0 {
		params["vpcId"] = vpcId
	}
	if len(peeringConnectionId) > 0 {
		params["peeringConnectionId"] = peeringConnectionId
	}
	body, err := region.vpc2017Request("DescribeVpcPeeringConnections", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, `region.vpcRequest("DescribeVpcPeeringConnections", %s)`, jsonutils.Marshal(params).String())
	}

	total, _ := body.Float("totalCount")
	if total <= 0 {
		return nil, int(total), nil
	}

	vpcPCs := make([]SVpcPC, 0)
	err = body.Unmarshal(&vpcPCs, "data")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "body.Unmarshal(&vpcPCs,%s)", body.String())
	}
	return vpcPCs, int(total), nil
}

func (region *SRegion) GetAllVpcPeeringConnections(vpcId string) ([]SVpcPC, error) {
	result := []SVpcPC{}
	for {
		vpcPCS, total, err := region.DescribeVpcPeeringConnections(vpcId, "", len(result), 50)
		if err != nil {
			return nil, errors.Wrapf(err, `client.DescribeVpcPeeringConnections(%s,"",%d,50)`, vpcId, len(result))
		}
		result = append(result, vpcPCS...)
		if total <= len(result) {
			break
		}
	}
	return result, nil
}

func (region *SRegion) GetVpcPeeringConnectionbyId(vpcPCId string) (*SVpcPC, error) {
	vpcPCs, _, err := region.DescribeVpcPeeringConnections("", vpcPCId, 0, 50)
	if err != nil {
		return nil, errors.Wrapf(err, `client.DescribeVpcPeeringConnections("", %s, 0, 50)`, vpcPCId)
	}
	if len(vpcPCs) < 1 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetVpcPeeringConnectionbyId(%s)", vpcPCId)
	}
	if len(vpcPCs) > 1 {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, "GetVpcPeeringConnectionbyId(%s)", vpcPCId)
	}
	return &vpcPCs[0], nil
}

func (region *SRegion) CreateVpcPeeringConnection(vpcId string, opts *cloudprovider.VpcPeeringConnectionCreateOptions) (string, error) {
	params := make(map[string]string)
	params["vpcId"] = vpcId
	params["peerVpcId"] = opts.PeerVpcId
	params["peerUin"] = opts.PeerAccountId
	params["peeringConnectionName"] = opts.Name
	body, err := region.vpc2017Request("CreateVpcPeeringConnection", params)
	if err != nil {
		return "", errors.Wrapf(err, `client.vpc2017Request("CreateVpcPeeringConnection", %s)`, jsonutils.Marshal(params).String())
	}
	peeringId, err := body.GetString("peeringConnectionId")
	if err != nil {
		return "", errors.Wrapf(err, `%s body.GetString("peeringConnectionId")`, body.String())
	}
	return peeringId, nil
}

func (region *SRegion) AcceptVpcPeeringConnection(peeringId string) error {
	params := make(map[string]string)
	params["peeringConnectionId"] = peeringId
	_, err := region.vpc2017Request("AcceptVpcPeeringConnection", params)
	if err != nil {
		return errors.Wrapf(err, `client.vpc2017Request("AcceptVpcPeeringConnection",%s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (region *SRegion) DeleteVpcPeeringConnection(peeringId string) error {
	params := make(map[string]string)
	params["peeringConnectionId"] = peeringId
	_, err := region.vpc2017Request("DeleteVpcPeeringConnection", params)
	if err != nil {
		return errors.Wrapf(err, `client.vpc2017Request("DeleteVpcPeeringConnection",%s)`, jsonutils.Marshal(params).String())
	}
	return nil
}

func (region *SRegion) CreateVpcPeeringConnectionEx(vpcId string, opts *cloudprovider.VpcPeeringConnectionCreateOptions) (int64, error) {
	params := make(map[string]string)
	params["vpcId"] = vpcId
	params["peerVpcId"] = opts.PeerVpcId
	params["peerUin"] = opts.PeerAccountId
	params["peeringConnectionName"] = opts.Name
	params["peerRegion"] = opts.PeerRegionId
	params["bandwidth"] = fmt.Sprintf("%d", opts.Bandwidth)
	body, err := region.vpc2017Request("CreateVpcPeeringConnectionEx", params)
	if err != nil {
		return 0, errors.Wrapf(err, `client.vpc2017Request("CreateVpcPeeringConnectionEx", %s)`, jsonutils.Marshal(params).String())
	}
	taskId, err := body.Int("taskId")
	if err != nil {
		return 0, errors.Wrapf(err, `%s body.Int("taskId")`, body.String())
	}
	return taskId, nil
}

func (region *SRegion) DescribeVpcTaskResult(taskId string) (int, error) {
	params := make(map[string]string)
	params["taskId"] = taskId
	body, err := region.vpc2017Request("DescribeVpcTaskResult", params)
	if err != nil {
		return 0, errors.Wrapf(err, `client.vpc2017Request("DescribeVpcTaskResult",%s)`, taskId)
	}
	status, err := body.Float("data", "status")
	if err != nil {
		return 0, errors.Wrapf(err, `%s body.Int("data.status")`, body.String())
	}
	return int(status), nil
}

func (region *SRegion) AcceptVpcPeeringConnectionEx(peeringId string) (int64, error) {
	params := make(map[string]string)
	params["peeringConnectionId"] = peeringId
	body, err := region.vpc2017Request("AcceptVpcPeeringConnectionEx", params)
	if err != nil {
		return 0, errors.Wrapf(err, `client.vpc2017Request("AcceptVpcPeeringConnectionEx", %s)`, jsonutils.Marshal(params).String())
	}
	taskId, err := body.Int("taskId")
	if err != nil {
		return 0, errors.Wrapf(err, `%s body.Int("taskId")`, body.String())
	}
	return taskId, nil
}

func (region *SRegion) DeleteVpcPeeringConnectionEx(peeringId string) (int64, error) {
	params := make(map[string]string)
	params["peeringConnectionId"] = peeringId
	body, err := region.vpc2017Request("DeleteVpcPeeringConnectionEx", params)
	if err != nil {
		return 0, errors.Wrapf(err, `client.vpc2017Request("DeleteVpcPeeringConnectionEx",%s)`, jsonutils.Marshal(params).String())
	}
	taskId, err := body.Int("taskId")
	if err != nil {
		return 0, errors.Wrapf(err, `%s body.Int("taskId")`, body.String())
	}
	return taskId, nil
}

func (self *SVpcPC) GetId() string {
	return self.PeeringConnectionID
}

func (self *SVpcPC) GetName() string {
	return self.PeeringConnectionName
}

func (self *SVpcPC) GetGlobalId() string {
	return self.GetId()
}

func (self *SVpcPC) GetStatus() string {
	switch self.State {
	case 1:
		return api.VPC_PEERING_CONNECTION_STATUS_ACTIVE
	case 0:
		return api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT
	default:
		return api.VPC_PEERING_CONNECTION_STATUS_UNKNOWN
	}
}

func (self *SVpcPC) Refresh() error {
	svpcPC, err := self.vpc.region.GetVpcPeeringConnectionbyId(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "self.vpc.region.GetVpcPeeringConnectionbyId(%s)", self.GetId())
	}
	return jsonutils.Update(self, svpcPC)
}

func (self *SVpcPC) GetPeerVpcId() string {
	return self.UnPeerVpcID
}

func (self *SVpcPC) GetPeerAccountId() string {
	if strings.Contains(self.PeerUin, ".") {
		return strings.Split(self.PeerUin, ".")[0]
	}
	return self.PeerUin
}

func (self *SVpcPC) GetEnabled() bool {
	return true
}

func (self *SVpcPC) Delete() error {
	if !self.IsCrossRegion() {
		return self.vpc.region.DeleteVpcPeeringConnection(self.GetId())
	}
	taskId, err := self.vpc.region.DeleteVpcPeeringConnectionEx(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "self.vpc.region.DeleteVpcPeeringConnection(%s)", self.GetId())
	}
	//err = self.vpc.region.WaitVpcTask(fmt.Sprintf("%d", taskId), time.Second*5, time.Minute*10)
	err = cloudprovider.Wait(time.Second*5, time.Minute*6, func() (bool, error) {
		status, err := self.vpc.region.DescribeVpcTaskResult(fmt.Sprintf("%d", taskId))
		if err != nil {
			return false, errors.Wrap(err, "self.vpc.region.DescribeVpcTaskResult")
		}
		//任务的当前状态。0：成功，1：失败，2：进行中。
		if status == 1 {
			return false, errors.Wrap(fmt.Errorf("taskfailed,taskId=%d", taskId), "client.DescribeVpcTaskResult(taskId)")
		}
		if status == 0 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return errors.Wrapf(err, "self.region.WaitTask(%d)", taskId)
	}
	return nil
}

func (self *SVpcPC) IsCrossRegion() bool {
	if len(self.Region) == 0 {
		return false
	}
	return self.Region != self.PeerRegion
}
