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
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/utils"
)

type SVerifyManager struct {
	SStatusStandaloneResourceBaseManager
}

var VerifyManager *SVerifyManager

func init() {
	VerifyManager = &SVerifyManager{
		SStatusStandaloneResourceBaseManager: NewStatusStandaloneResourceBaseManager(
			SVerify{},
			"notify_t_verify",
			"verification",
			"verifications",
		),
	}
	VerifyManager.SetVirtualObject(VerifyManager)
}

type SVerify struct {
	SStatusStandaloneResourceBase

	CID      string    `width:"128" nullable:"false" create:"required" list:"user"`
	Token    string    `width:"200" nullable:"false" create:"required" list:"user"`
	SendAt   time.Time `nullable:"true" create:"optional"`
	ExpireAt time.Time `create:"required" list:"user"`
}

// NewSVerify Generate a SVerify instance which implement a Verification Token.
func NewSVerify(contactType string, cid string) *SVerify {
	var token string
	var expireAt time.Time
	now := time.Now()
	if contactType == EMAIL {
		token = utils.GenerateEmailToken(32)
		expireAt = now.Add(12 * time.Hour)
	} else {
		token = utils.GenerateMobileToken()
		expireAt = now.Add(2 * time.Minute)
	}
	ret := &SVerify{
		CID:      cid,
		Token:    token,
		ExpireAt: expireAt,
		SendAt:   now,
	}
	ret.ID = DefaultUUIDGenerator()
	ret.SetModelManager(VerifyManager, ret)
	return ret
}

func (self *SVerifyManager) InitializeData() error {
	sql := fmt.Sprintf("update %s set updated_at=update_at, deleted=is_deleted", self.TableSpec().Name())
	q := sqlchemy.NewRawQuery(sql, "")
	q.Row()
	return nil
}

func (self *SVerifyManager) FetchByCID(cid string, filter func(q *sqlchemy.SQuery) *sqlchemy.SQuery) ([]SVerify, error) {
	q := self.Query()
	q.Filter(sqlchemy.Equals(q.Field("cid"), cid))
	q = filter(q)
	records := make([]SVerify, 0, 1)
	err := db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (self *SVerifyManager) FetchByID(id string) ([]SVerify, error) {
	q := self.Query()
	q.Filter(sqlchemy.Equals(q.Field("id"), id))
	records := make([]SVerify, 0, 1)
	err := db.FetchModelObjects(self, q, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (self *SVerifyManager) Create(ctx context.Context, userCred mcclient.TokenCredential, verify *SVerify) error {
	data := jsonutils.Marshal(verify)
	ownerID, err := utils.FetchOwnerId(ctx, self, userCred, data)
	if err != nil {
		return err
	}
	_, err = db.DoCreate(self, ctx, userCred, jsonutils.JSONNull, data, ownerID)
	if err != nil {
		return err
	}
	return nil
}
