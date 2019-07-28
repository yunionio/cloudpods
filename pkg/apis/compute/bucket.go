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

package compute

const (
	BUCKET_OPS_STATS_CHANGE = "stats_change"

	BUCKET_STATUS_START_CREATE = "start_create"
	BUCKET_STATUS_CREATING     = "creating"
	BUCKET_STATUS_READY        = "ready"
	BUCKET_STATUS_CREATE_FAIL  = "create_fail"
	BUCKET_STATUS_START_DELETE = "start_delete"
	BUCKET_STATUS_DELETING     = "deleting"
	BUCKET_STATUS_DELETED      = "deleted"
	BUCKET_STATUS_DELETE_FAIL  = "delete_fail"

	BUCKET_UPLOAD_OBJECT_KEY_HEADER          = "X-Yunion-Bucket-Upload-Key"
	BUCKET_UPLOAD_OBJECT_STORAGECLASS_HEADER = "X-Yunion-Bucket-Upload-Storageclass"
)
