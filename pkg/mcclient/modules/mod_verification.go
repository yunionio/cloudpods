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

var (
	Verifications modulebase.ResourceManager
)

func init() {
	Verifications = NewNotifyManager("verification", "verifications",
		[]string{"id", "cid", "sent_at", "expire_at", "status", "create_at", "update_at", "delete_at", "create_by", "update_by", "delete_by", "is_deleted", "remark"},
		[]string{})

	register(&Verifications)
}
