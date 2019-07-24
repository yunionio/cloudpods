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

import (
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SBucketManager struct {
	ResourceManager
}

func (manager *SBucketManager) Upload(s *mcclient.ClientSession, bucketId string, key string, body io.Reader, contType string, storageClass string) error {
	method := httputils.POST
	path := fmt.Sprintf("/%s/%s/upload", manager.URLPath(), bucketId)
	headers := http.Header{}
	headers.Set(api.BUCKET_UPLOAD_OBJECT_KEY_HEADER, key)
	if len(contType) > 0 {
		headers.Set("Content-Type", contType)
	}
	if len(storageClass) > 0 {
		headers.Set(api.BUCKET_UPLOAD_OBJECT_STORAGECLASS_HEADER, storageClass)
	}

	_, err := manager.rawRequest(s, method, path, headers, body)
	if err != nil {
		return errors.Wrap(err, "rawRequest")
	}
	return nil
}

var (
	Buckets SBucketManager
)

func init() {
	Buckets = SBucketManager{
		NewComputeManager("bucket", "buckets",
			[]string{"ID", "Name", "Storage_Class",
				"Status", "location", "acl",
				"region", "manager_id",
			},
			[]string{}),
	}

	registerCompute(&Buckets)
}
