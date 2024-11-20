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

package baidu

import (
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"
)

type SSnapshotPolicy struct {
	multicloud.SVirtualResourceBase
	SBaiduTag
	region *SRegion

	Id              string
	Name            string
	TimePoints      []int
	RepeatWeekdays  []int
	RetentionDays   int
	Status          string
	CreatedTime     time.Time
	UpdatedTime     time.Time
	DeletedTime     time.Time
	LastExecuteTime time.Time
	VolumeCount     int
}

func (self *SSnapshotPolicy) GetId() string {
	return self.Id
}

func (self *SSnapshotPolicy) GetName() string {
	return self.Name
}

func (self *SSnapshotPolicy) GetStatus() string {
	switch self.Status {
	case "active":
		return apis.STATUS_AVAILABLE
	case "deleted":
		return apis.STATUS_DELETING
	case "paused":
		return apis.STATUS_UNKNOWN
	default:
		return apis.STATUS_UNKNOWN
	}
}

func (self *SSnapshotPolicy) Refresh() error {
	policy, err := self.region.GetSnapshotPolicy(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, policy)
}

func (self *SSnapshotPolicy) GetGlobalId() string {
	return self.Id
}

func (self *SSnapshotPolicy) GetRetentionDays() int {
	return self.RetentionDays
}

func (self *SSnapshotPolicy) GetRepeatWeekdays() ([]int, error) {
	return self.RepeatWeekdays, nil
}

func (self *SSnapshotPolicy) GetTimePoints() ([]int, error) {
	return self.TimePoints, nil
}

func (self *SSnapshotPolicy) Delete() error {
	return self.region.DeletSnapshotPolicy(self.Id)
}

func (self *SSnapshotPolicy) ApplyDisks(ids []string) error {
	return self.region.ApplySnapshotPolicy(self.Id, ids)
}

func (self *SSnapshotPolicy) CancelDisks(ids []string) error {
	return self.region.CancelSnapshotPolicy(self.Id, ids)
}

func (self *SSnapshotPolicy) GetApplyDiskIds() ([]string, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (region *SRegion) GetSnapshotPolicys() ([]SSnapshotPolicy, error) {
	params := url.Values{}
	ret := []SSnapshotPolicy{}
	for {
		resp, err := region.bccList("v2/asp", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			NextMarker          string
			AutoSnapshotPolicys []SSnapshotPolicy
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.AutoSnapshotPolicys...)
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return ret, nil
}

func (region *SRegion) GetSnapshotPolicy(id string) (*SSnapshotPolicy, error) {
	params := url.Values{}
	resp, err := region.bccList(fmt.Sprintf("v2/asp/%s", id), params)
	if err != nil {
		return nil, err
	}
	ret := &SSnapshotPolicy{region: region}
	err = resp.Unmarshal(ret, "autoSnapshotPolicy")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *SRegion) CreateSnapshotPolicy(opts *cloudprovider.SnapshotPolicyInput) (string, error) {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	body := map[string]interface{}{
		"name":           opts.Name,
		"timePoints":     opts.TimePoints,
		"repeatWeekdays": opts.RetentionDays,
		"retentionDays":  opts.RetentionDays,
	}
	resp, err := r.bccPost("v2/asp", params, body)
	if err != nil {
		return "", err
	}
	return resp.GetString("aspId")
}

func (r *SRegion) GetISnapshotPolicyById(id string) (cloudprovider.ICloudSnapshotPolicy, error) {
	policy, err := r.GetSnapshotPolicy(id)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

func (self *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	policys, err := self.GetSnapshotPolicys()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSnapshotPolicy{}
	for i := range policys {
		policys[i].region = self
		ret = append(ret, &policys[i])
	}
	return ret, nil
}

func (self *SRegion) DeletSnapshotPolicy(id string) error {
	_, err := self.bccList(fmt.Sprintf("v2/asp/%s", id), nil)
	return err
}

func (self *SRegion) ApplySnapshotPolicy(id string, diskIds []string) error {
	params := url.Values{}
	params.Set("attach", "")
	body := map[string]interface{}{
		"volumeIds": diskIds,
	}
	_, err := self.bccUpdate(fmt.Sprintf("v2/asp/%s", id), params, body)
	return err
}

func (self *SRegion) CancelSnapshotPolicy(id string, diskIds []string) error {
	params := url.Values{}
	params.Set("detach", "")
	body := map[string]interface{}{
		"volumeIds": diskIds,
	}
	_, err := self.bccUpdate(fmt.Sprintf("v2/asp/%s", id), params, body)
	return err
}
