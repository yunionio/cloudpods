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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SEventManager struct {
	db.SStandaloneAnonResourceBaseManager
}

var EventManager *SEventManager

func init() {
	EventManager = &SEventManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			SEvent{},
			"events_tbl",
			"notifyevent",
			"notifyevents",
		),
	}
	EventManager.SetVirtualObject(EventManager)
}

type SEvent struct {
	db.SStandaloneAnonResourceBase

	Message     string
	Event       string `width:"64" nullable:"true"`
	AdvanceDays int
}

func (e *SEventManager) CreateEvent(ctx context.Context, event, message string, advanceDays int) (*SEvent, error) {
	eve := &SEvent{
		Message:     message,
		Event:       event,
		AdvanceDays: advanceDays,
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
