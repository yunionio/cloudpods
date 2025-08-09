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

package taskman

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type STaskBase struct {
	db.SStatusResourceBase

	// 开始任务时间
	StartAt time.Time `nullable:"true" list:"user" json:"start_at"`
	// 完成任务时间
	EndAt time.Time `nullable:"true" list:"user" json:"end_at"`

	ObjType  string `old_name:"obj_name" json:"obj_type" width:"128" charset:"utf8" nullable:"false" default:"" list:"user"`
	Object   string `json:"object" width:"128" charset:"utf8" nullable:"false" default:"" list:"user"` //  Column(VARCHAR(128, charset='utf8'), nullable=False)
	ObjId    string `width:"128" charset:"ascii" nullable:"false" list:"user" index:"true"`            // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
	TaskName string `width:"64" charset:"ascii" nullable:"false" list:"user" index:"true"`             // Column(VARCHAR(64, charset='ascii'), nullable=False)

	UserCred mcclient.TokenCredential `width:"1024" charset:"utf8" nullable:"false" get:"user"` // Column(VARCHAR(1024, charset='ascii'), nullable=False)
	// OwnerCred string `width:"512" charset:"ascii" nullable:"true"` // Column(VARCHAR(512, charset='ascii'), nullable=True)
	Params *jsonutils.JSONDict `charset:"utf8" length:"medium" nullable:"false" get:"user"` // Column(MEDIUMTEXT(charset='ascii'), nullable=False)

	Stage string `width:"64" charset:"ascii" nullable:"false" default:"on_init" list:"user" index:"true"` // Column(VARCHAR(64, charset='ascii'), nullable=False, default='on_init')

	// 父任务Id
	ParentTaskId string `width:"36" charset:"ascii" list:"user" index:"true" json:"parent_task_id"`
}
