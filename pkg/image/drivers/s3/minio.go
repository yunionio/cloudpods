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

package s3

import (
	"context"
	"fmt"
	"io"
	"os"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/image"
)

type Err string

func (e Err) Error() string {
	return string(e)
}

const ErrClientNotInit = Err("s3 client not init")

var client *S3Client

// Bucket is image upload bucket name
type S3Client struct {
	osc      *objectstore.SObjectStoreClient
	bucket   string
	endpoint string
}

func (c *S3Client) Location(filePath string) string {
	return fmt.Sprintf("%s%s", image.S3Prefix, filePath)
}

func (c *S3Client) getBucket() (cloudprovider.ICloudBucket, error) {
	return c.osc.GetIBucketByName(c.bucket)
}

func Init(endpoint, accessKey, secretKey, bucket string, useSSL bool) error {
	if client != nil {
		return nil
	}
	cfg := objectstore.NewObjectStoreClientConfig(endpoint, accessKey, secretKey)
	minioClient, err := objectstore.NewObjectStoreClient(cfg)
	if err != nil {
		return errors.Wrap(err, "new minio client")
	}
	client = &S3Client{
		osc:      minioClient,
		bucket:   bucket,
		endpoint: endpoint,
	}
	err = ensureBucket()
	if err != nil {
		return errors.Wrap(err, "ensure bucket")
	}
	return nil
}

func ensureBucket() error {
	exists, err := client.osc.IBucketExist(client.bucket)
	if err != nil {
		return errors.Wrap(err, "call bucket exists")
	}
	if !exists {
		if err = client.osc.CreateIBucket(client.bucket, "", "private"); err != nil {
			return errors.Wrap(err, "call make bucket")
		}
	}
	return nil
}

func Put(ctx context.Context, filePath, objName string) (string, error) {
	if client == nil {
		return "", ErrClientNotInit
	}
	bucket, err := client.getBucket()
	if err != nil {
		return "", errors.Wrap(err, "client.getBucket")
	}

	finfo, err := os.Stat(filePath)
	if err != nil {
		return "", errors.Wrap(err, "os.Stat")
	}
	fSize := finfo.Size()
	file, err := os.Open(filePath)
	if err != nil {
		return "", errors.Wrap(err, "os.Open")
	}
	defer file.Close()
	const blockSizeMB = 100
	err = cloudprovider.UploadObject(ctx, bucket, objName, blockSizeMB*1000*1000, file, fSize, cloudprovider.ACLPrivate, "", nil, false)
	if err != nil {
		return "", errors.Wrap(err, "cloudprovider.UploadObject")
	}
	log.Debugf("put object %s size %d", objName, fSize)
	return client.Location(objName), nil
}

func Get(ctx context.Context, fileName string) (int64, io.ReadCloser, error) {
	if client == nil {
		return 0, nil, ErrClientNotInit
	}

	bucket, err := client.getBucket()
	if err != nil {
		return 0, nil, errors.Wrap(err, "client.getBucket")
	}
	result, err := bucket.ListObjects(fileName, "", "", 1)
	if err != nil {
		return 0, nil, errors.Wrap(err, "bucket.ListObject")
	}

	rc, err := bucket.GetObject(ctx, fileName, nil)
	if err != nil {
		return 0, nil, errors.Wrap(err, "bucket.GetObject")
	}

	return result.Objects[0].GetSizeBytes(), rc, err
}

func Remove(ctx context.Context, fileName string) error {
	if client == nil {
		return ErrClientNotInit
	}

	bucket, err := client.getBucket()
	if err != nil {
		return errors.Wrap(err, "client.getBucket")
	}

	return bucket.DeleteObject(ctx, fileName)
}
