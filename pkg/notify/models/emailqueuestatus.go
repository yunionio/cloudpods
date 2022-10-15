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
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SEmailQueueStatusManager struct {
	db.SModelBaseManager
}

type SEmailQueueStatus struct {
	db.SModelBase

	Id int64 `primary:"true" list:"user"`

	SentAt time.Time `list:"user"`

	Status string `width:"16" charset:"ascii" default:"queued" list:"user"`

	Results string `list:"user" charset:"utf8"`
}

var EmailQueueStatusManager *SEmailQueueStatusManager

func init() {
	EmailQueueStatusManager = &SEmailQueueStatusManager{
		SModelBaseManager: db.NewModelBaseManager(SEmailQueueStatus{}, "emailqueue_status_tbl", "emailqueue_status", "emailqueue_status"),
	}
	EmailQueueStatusManager.SetVirtualObject(EmailQueueStatusManager)
}

func (manager *SEmailQueueStatusManager) fetchEmailQueueStatus(ids []int64) (map[int64]SEmailQueueStatus, error) {
	q := manager.Query().In("id", ids)
	results := make([]SEmailQueueStatus, 0)
	err := q.All(&results)
	if err != nil {
		return nil, errors.Wrap(err, "query.All")
	}
	ret := make(map[int64]SEmailQueueStatus)
	for i := range results {
		eqs := results[i]
		ret[eqs.Id] = eqs
	}
	return ret, nil
}
