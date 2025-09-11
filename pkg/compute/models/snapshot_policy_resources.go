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

package models

import (
	"fmt"

	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SSnapshotPolicyResourceManager struct {
	db.SResourceBaseManager
}

var SnapshotPolicyResourceManager *SSnapshotPolicyResourceManager

func init() {
	SnapshotPolicyResourceManager = &SSnapshotPolicyResourceManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SSnapshotPolicyResource{},
			"snapshot_policy_resources_tbl",
			"snapshot_policy_resource",
			"snapshot_policy_resources",
		),
	}
	SnapshotPolicyResourceManager.SetVirtualObject(SnapshotPolicyResourceManager)
}

type SSnapshotPolicyResource struct {
	db.SResourceBase

	SnapshotpolicyId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	ResourceId       string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	ResourceType     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (man *SSnapshotPolicyResourceManager) RemoveByResource(id, typ string) error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where resource_id = ? and resource_type = ?",
			man.TableSpec().Name(),
		), id, typ,
	)
	return err
}

func (self *SSnapshotPolicyResource) GetServer() (*SGuest, error) {
	guest, err := GuestManager.FetchById(self.ResourceId)
	if err != nil {
		return nil, err
	}
	return guest.(*SGuest), nil
}

func (man *SSnapshotPolicyResourceManager) RemoveBySnapshotpolicy(id string) error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where snapshotpolicy_id = ?",
			man.TableSpec().Name(),
		), id,
	)
	return err
}
