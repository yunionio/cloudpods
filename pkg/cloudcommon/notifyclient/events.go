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

package notifyclient

const (
	SYSTEM_ERROR   = "SYSTEM_ERROR"
	SYSTEM_WARNING = "SYSTEM_WARNING"

	SERVER_CREATED       = "SERVER_CREATED"
	SERVER_CREATED_ADMIN = "SERVER_CREATED_ADMIN"
	SERVER_DELETED       = "SERVER_DELETED"
	SERVER_DELETED_ADMIN = "SERVER_DELETED_ADMIN"
	SERVER_REBUILD_ROOT  = "SERVER_REBUILD_ROOT"
	SERVER_CHANGE_FLAVOR = "SERVER_CHANGE_FLAVOR"
	SERVER_PANICKED      = "SERVER_PANICKED"

	IMAGE_ACTIVED = "IMAGE_ACTIVED"
)
