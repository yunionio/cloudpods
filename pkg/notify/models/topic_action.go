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
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type STopicActionManager struct {
	db.SResourceBaseManager
}

var TopicActionManager *STopicActionManager

func init() {
	TopicActionManager = &STopicActionManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			STopicAction{},
			"topic_actions_tbl",
			"topic_action",
			"topic_actions",
		),
	}
	TopicActionManager.SetVirtualObject(TopicActionManager)
	TopicActionManager.TableSpec().AddIndex(false, "topic_id", "action_id", "deleted")
}

type STopicAction struct {
	db.SResourceBase

	ActionId string `width:"64" nullable:"false" create:"required" update:"user" list:"user"`
	TopicId  string `width:"64" nullable:"false" create:"required" update:"user" list:"user"`
}
