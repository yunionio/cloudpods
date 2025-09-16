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
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	_ "yunion.io/x/cloudmux/pkg/multicloud/objectstore/provider"
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

func GetIBucket(ctx context.Context, sess *mcclient.ClientSession, bucketDetails *api.BucketDetails) (cloudprovider.ICloudBucket, error) {
	provider, err := Cloudproviders.GetProvider(ctx, sess, bucketDetails.ManagerId)
	if err != nil {
		return nil, errors.Wrap(err, "computemodules.Cloudproviders.GetProvider")
	}

	iregion, err := func() (cloudprovider.ICloudRegion, error) {
		if provider.GetFactory().IsOnPremise() {
			return provider.GetOnPremiseIRegion()
		} else {
			return provider.GetIRegionById(bucketDetails.RegionExternalId)
		}
	}()
	if err != nil {
		return nil, errors.Wrap(err, "GetIRegion")
	}

	bucket, err := iregion.GetIBucketById(bucketDetails.ExternalId)
	if err != nil {
		return nil, errors.Wrap(err, "iregion.GetIBucketById")
	}

	return bucket, nil
}

type nullWriter struct{}

func (w *nullWriter) WriteAt(p []byte, off int64) (n int, err error) {
	return len(p), nil
}

var (
	randBuffer [1024]byte
)

func init() {
	for i := range randBuffer {
		randBuffer[i] = byte(rand.Intn(256))
	}
}

type randReader struct {
	offset    int64
	sizeBytes int64
}

func newRandReader(sizeBytes int64) io.Reader {
	return &randReader{offset: 0, sizeBytes: sizeBytes}
}

func (r *randReader) Read(p []byte) (n int, err error) {
	if r.offset >= r.sizeBytes {
		return 0, io.EOF
	}
	offset := 0
	for offset < len(p) && r.offset+int64(offset) < r.sizeBytes {
		readLen := len(randBuffer)
		if readLen > len(p)-offset {
			readLen = len(p) - offset
		}
		if readLen > int(r.sizeBytes-r.offset) {
			readLen = int(r.sizeBytes-r.offset) - offset
		}
		copy(p[offset:], randBuffer[:readLen])
		offset += readLen
	}
	r.offset += int64(offset)
	return offset, nil
}

func ProbeBucketStats(ctx context.Context, bucket cloudprovider.ICloudBucket, testKey string, sizeBytes int64) (*api.BucketProbeResult, error) {
	result := &api.BucketProbeResult{}
	start := time.Now()

	// force upload object in one shot
	err := cloudprovider.UploadObject(ctx, bucket, testKey, sizeBytes*2, newRandReader(sizeBytes), sizeBytes, cloudprovider.ACLPrivate, "", nil, false)
	if err != nil {
		return nil, errors.Wrap(err, "cloudprovider.UploadObject")
	}

	result.UploadTime = time.Since(start)

	_, err = cloudprovider.DownloadObjectParallel(ctx, bucket, testKey, nil, &nullWriter{}, 0, 0, false, 1)
	if err != nil {
		return nil, errors.Wrap(err, "cloudprovider.DownloadObjectParallel")
	}

	result.DownloadTime = time.Since(start) - result.UploadTime

	err = bucket.DeleteObject(ctx, testKey)
	if err != nil {
		return nil, errors.Wrap(err, "bucket.DeleteObject")
	}

	result.DeleteTime = time.Since(start) - result.DownloadTime

	return result, nil
}
