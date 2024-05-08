// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http//www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type STopicActionManager struct {
	db.SJointResourceBaseManager
}

var TopicActionManager *STopicActionManager

func init() {
	TopicActionManager = &STopicActionManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			STopicAction{},
			"topic_actions_tbl",
			"topic_action",
			"topic_actions",
			TopicManager,
			NotifyActionManager,
		),
	}
	TopicActionManager.SetVirtualObject(TopicActionManager)
}

type STopicAction struct {
	db.SJointResourceBase

	ActionId string `width:"64" nullable:"false" create:"required" update:"user" list:"user"`
	TopicId  string `width:"64" nullable:"false" create:"required" update:"user" list:"user"`
}

func (manager *STopicActionManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.STopicActionCreateInput) (api.STopicActionCreateInput, error) {
	count, err := TopicManager.Query().Equals("id", input.TopicId).CountWithError()
	if err != nil {
		return input, errors.Wrap(err, "fetch count")
	}
	if count == 0 {
		return input, errors.Wrap(errors.ErrNotFound, "topic_id")
	}
	count, err = NotifyActionManager.Query().Equals("id", input.ActionId).CountWithError()
	if err != nil {
		return input, errors.Wrap(err, "fetch count")
	}
	if count == 0 {
		return input, errors.Wrap(errors.ErrNotFound, "action_id")
	}
	count, err = manager.Query().Equals("action_id", input.ActionId).Equals("topic_id", input.TopicId).CountWithError()
	if err != nil {
		return input, errors.Wrap(err, "fetch count")
	}
	if count > 0 {
		return input, errors.Wrapf(httperrors.ErrDuplicateResource, "topic:%s,action:%s has been exist", input.TopicId, input.ActionId)
	}
	return input, nil
}
