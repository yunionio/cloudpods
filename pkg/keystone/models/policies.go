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
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	policyman "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SPolicyManager struct {
	db.SStandaloneResourceBaseManager
}

var PolicyManager *SPolicyManager

func init() {
	PolicyManager = &SPolicyManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SPolicy{},
			"policy",
			"policy",
			"policies",
		),
	}
}

/*
+-------+--------------+------+-----+---------+-------+
| Field | Type         | Null | Key | Default | Extra |
+-------+--------------+------+-----+---------+-------+
| id    | varchar(64)  | NO   | PRI | NULL    |       |
| type  | varchar(255) | NO   |     | NULL    |       |
| blob  | text         | NO   |     | NULL    |       |
| extra | text         | YES  |     | NULL    |       |
+-------+--------------+------+-----+---------+-------+
*/

type SPolicy struct {
	db.SStandaloneResourceBase

	Type string `width:"255" charset:"utf8" nullable:"false" list:"user" update:"admin"`

	Blob jsonutils.JSONObject `nullable:"false" list:"user" update:"admin"`

	Extra *jsonutils.JSONDict `nullable:"true" list:"user"`

	Enabled tristate.TriState `nullable:"false" default:"true" list:"admin" update:"admin" create:"admin_optional"`
}

func (manager *SPolicyManager) InitializeData() error {
	q := manager.Query()
	q = q.IsNullOrEmpty("name")
	policies := make([]SPolicy, 0)
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil {
		return err
	}
	for i := range policies {
		db.Update(&policies[i], func() error {
			policies[i].Name = policies[i].Type
			policies[i].Description, _ = policies[i].Extra.GetString("description")
			return nil
		})
	}
	return nil
}

func (manager *SPolicyManager) FetchEnabledPolicies() ([]SPolicy, error) {
	q := manager.Query().IsTrue("enabled")

	policies := make([]SPolicy, 0)
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return policies, nil
}

func (manager *SPolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	typeStr, _ := data.GetString("type")
	if len(typeStr) == 0 {
		return nil, httperrors.NewInputParameterError("missing input field type")
	}
	if !data.Contains("name") {
		data.Set("name", jsonutils.NewString(typeStr))
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (policy *SPolicy) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	policy.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	policy.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	policyman.PolicyManager.SyncOnce()
}

func (policy *SPolicy) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	policy.SStandaloneResourceBase.PostDelete(ctx, userCred)
	policyman.PolicyManager.SyncOnce()
}
