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

package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

type NoticeManager struct {
	modulebase.ResourceManager
}

type NoticeReadMarkManager struct {
	modulebase.ResourceManager
}

var (
	Notice         NoticeManager
	NoticeReadMark NoticeReadMarkManager
)

func init() {
	Notice = NoticeManager{NewYunionAgentManager("notice", "notices",
		[]string{"id", "created_at", "updated_at", "author_id", "author", "title", "content"},
		[]string{})}
	register(&Notice)

	NoticeReadMark = NoticeReadMarkManager{NewYunionAgentManager("readmark", "readmarks",
		[]string{},
		[]string{"notice_id", "user_id"})}
	register(&NoticeReadMark)
}
