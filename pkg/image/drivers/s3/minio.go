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
	"fmt"

	"github.com/minio/minio-go"

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
	*minio.Client
	bucket   string
	endpoint string
}

func (c *S3Client) Location(filePath string) string {
	return fmt.Sprintf("%s%s", image.S3Prefix, filePath)
}

func Init(endpoint, accessKey, secretKey, bucket string, useSSL bool) error {
	if client != nil {
		return nil
	}
	minioClient, err := minio.New(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		return errors.Wrap(err, "new minio client")
	}
	client = &S3Client{
		Client:   minioClient,
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
	exists, err := client.BucketExists(client.bucket)
	if err != nil {
		return errors.Wrap(err, "call bucket exists")
	}
	if !exists {
		if err = client.MakeBucket(client.bucket, ""); err != nil {
			return errors.Wrap(err, "call make bucket")
		}
	}
	return nil
}

func Put(filePath, objName string) (string, error) {
	if client == nil {
		return "", ErrClientNotInit
	}

	size, err := client.FPutObject(client.bucket, objName, filePath, minio.PutObjectOptions{})
	if err != nil {
		return "", errors.Wrap(err, "put object")
	}
	log.Debugf("put object %s size %d", objName, size)
	return client.Location(objName), nil
}

func Get(fileName string) (*minio.Object, error) {
	if client == nil {
		return nil, ErrClientNotInit
	}
	obj, err := client.GetObject(client.bucket, fileName, minio.GetObjectOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "get object %s", fileName)
	}
	return obj, nil
}

func Remove(fileName string) error {
	if client == nil {
		return ErrClientNotInit
	}
	return client.RemoveObject(client.bucket, fileName)
}
