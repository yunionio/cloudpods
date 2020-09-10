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
	CACHED_IMAGE_STATUS_INIT         = "init"
	CACHED_IMAGE_STATUS_SAVING       = "saving"
	CACHED_IMAGE_STATUS_CACHING      = "caching"
	CACHED_IMAGE_STATUS_ACTIVE       = "active"
	CACHED_IMAGE_STATUS_DELETING     = "deleting"
	CACHED_IMAGE_STATUS_CACHE_FAILED = "cache_fail"

	DOWNLOAD_SESSION_LENGTH = 3600 * 3 // 3 hour
)

const (
	CACHED_IMAGE_REFRESH_SECONDS                  = 900   // 15 minutes
	CACHED_IMAGE_REFERENCE_SESSION_EXPIRE_SECONDS = 86400 // 1 day
)
