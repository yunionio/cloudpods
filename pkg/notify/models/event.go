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
	"time"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SEventManager struct {
	db.SLogBaseManager
}

var EventManager *SEventManager

func InitEventLog() {
	EventManager = &SEventManager{
		SLogBaseManager: db.NewLogBaseManager(SEvent{}, "events2_tbl", "notifyevent", "notifyevents", "created_at", consts.OpsLogWithClickhouse),
	}
	EventManager.SetVirtualObject(EventManager)
}

type SEvent struct {
	db.SLogBase

	// 资源创建时间
	CreatedAt time.Time `nullable:"false" created_at:"true" index:"true" get:"user" list:"user" json:"created_at"`

	Message      string
	Event        string `width:"64" nullable:"true"`
	ResourceType string `width:"64" nullable:"true"`
	Action       string `width:"64" nullable:"true"`
	AdvanceDays  int
	TopicId      string `width:"128" nullable:"true" index:"true"`
}

func (e *SEventManager) CreateEvent(ctx context.Context, event, topicId, message, action, resourceType string, advanceDays int) (*SEvent, error) {
	eve := &SEvent{
		Message:      message,
		Event:        event,
		AdvanceDays:  advanceDays,
		TopicId:      topicId,
		Action:       action,
		ResourceType: resourceType,
	}
	err := e.TableSpec().Insert(ctx, eve)
	if err != nil {
		return nil, err
	}
	return eve, nil
}

func (e *SEventManager) GetEvent(id string) (*SEvent, error) {
	model, err := e.FetchById(id)
	if err != nil {
		return nil, err
	}
	return model.(*SEvent), nil
}

func (e *SEvent) GetRecordTime() time.Time {
	return e.CreatedAt
}
