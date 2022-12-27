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

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SBucketManager struct {
	modulebase.ResourceManager
}

func (manager *SBucketManager) Upload(s *mcclient.ClientSession, bucketId string, key string, body io.Reader, contLength int64, storageClass string, acl string, meta http.Header) error {
	method := httputils.POST
	path := fmt.Sprintf("/%s/%s/upload", manager.URLPath(), bucketId)
	headers := cloudprovider.MetaToHttpHeader(cloudprovider.META_HEADER_PREFIX, meta)
	headers.Set(api.BUCKET_UPLOAD_OBJECT_KEY_HEADER, key)

	if contLength > 0 {
		headers.Set("Content-Length", strconv.FormatInt(contLength, 10))
	}

	if len(storageClass) > 0 {
		headers.Set(api.BUCKET_UPLOAD_OBJECT_STORAGECLASS_HEADER, storageClass)
	}

	if len(acl) > 0 {
		headers.Set(api.BUCKET_UPLOAD_OBJECT_ACL_HEADER, acl)
	}

	resp, err := modulebase.RawRequest(manager.ResourceManager, s, method, path, headers, body)
	if err != nil {
		return errors.Wrap(err, "rawRequest")
	}

	_, _, err = s.ParseJSONResponse("", resp, err)
	if err != nil {
		return err
	}

	return nil
}

var (
	Buckets SBucketManager
)

func init() {
	Buckets = SBucketManager{
		modules.NewComputeManager("bucket", "buckets",
			[]string{"ID", "Name", "Storage_Class",
				"Status", "location", "acl",
				"region", "manager_id", "public_scope",
			},
			[]string{}),
	}

	modules.RegisterCompute(&Buckets)
}
