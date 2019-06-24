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

package multicloud

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRegion struct{}

func (r *SRegion) CreateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput) (string, error) {
	return "", fmt.Errorf("CreateSnapshotPolicy not implement")
}

func (r *SRegion) GetISnapshotPolicyById(snapshotPolicyId string) (cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, fmt.Errorf("GetISnapshotPolicyById not implement")
}

func (self *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, fmt.Errorf("GetISnapshotPolicies not implement")
}

func (self *SRegion) DeleteSnapshotPolicy(string) error {
	return fmt.Errorf("DeleteSnapshotPolicy not implement")
}

func (self *SRegion) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskIds []string) error {
	return fmt.Errorf("ApplySnapshotPolicyToDisks not implement")
}

func (self *SRegion) CancelSnapshotPolicyToDisks(diskIds []string) error {
	return fmt.Errorf("ApplySnapshotPolicyToDisks not implement")
}

func (self *SRegion) GetISkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) CreateISku(*cloudprovider.SServerSku) (cloudprovider.ICloudSku, error) {
	return nil, fmt.Errorf("Not Support Create Server Sku")
}

func (self *SRegion) GetISkuById(skuId string) (cloudprovider.ICloudSku, error) {
	return nil, fmt.Errorf("Not Support GetISkuById")
}
