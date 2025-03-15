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

import "yunion.io/x/cloudmux/pkg/apis/compute"

const (
	CACHED_IMAGE_STATUS_INIT                 = compute.CACHED_IMAGE_STATUS_INIT
	CACHED_IMAGE_STATUS_SAVING               = compute.CACHED_IMAGE_STATUS_SAVING
	CACHED_IMAGE_STATUS_CACHING              = compute.CACHED_IMAGE_STATUS_CACHING
	CACHED_IMAGE_STATUS_ACTIVE               = compute.CACHED_IMAGE_STATUS_ACTIVE
	CACHED_IMAGE_STATUS_DELETING             = compute.CACHED_IMAGE_STATUS_DELETING
	CACHED_IMAGE_STATUS_DELETE_FAILED        = "delete_failed"
	CACHED_IMAGE_STATUS_UNCACHE_IMAGE_FAILED = "uncache_image_fail"
	CACHED_IMAGE_STATUS_CACHE_FAILED         = compute.CACHED_IMAGE_STATUS_CACHE_FAILED
	CACHED_IMAGE_STATUS_UNKNOWN              = compute.CACHED_IMAGE_STATUS_UNKNOWN

	DOWNLOAD_SESSION_LENGTH = 3600 * 3 // 3 hour
)

const (
	CACHED_IMAGE_REFRESH_SECONDS                  = 1     // 1 second
	CACHED_IMAGE_REFERENCE_SESSION_EXPIRE_SECONDS = 86400 // 1 day
)
