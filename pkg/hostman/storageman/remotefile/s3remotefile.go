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

package remotefile

import (
	"context"
	"os"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type S3RemoteFileInfo struct {
	Bucket   string `json:"bucket"`
	AcessKey string `json:"access_key"`
	Secret   string `json:"secret"`
	Url      string `json:"url"`
	Key      string `json:"key"`
	SignVer  string `json:"sign_ver"`
}

func (info *S3RemoteFileInfo) download(ctx context.Context, localPath string, callback func(progress, progressMbps float64, totalSizeMb int64)) error {
	log.Infof("start s3 download url: %s key: %s to %s", info.Url, info.Key, localPath)

	cfg := objectstore.NewObjectStoreClientConfig(info.Url, info.AcessKey, info.Secret)
	if len(info.SignVer) > 0 {
		cfg.SignVersion(objectstore.S3SignVersion(info.SignVer))
	}
	minioClient, err := objectstore.NewObjectStoreClient(cfg)
	if err != nil {
		return errors.Wrap(err, "new minio client")
	}
	bucket, err := minioClient.GetIBucketByName(info.Bucket)
	if err != nil {
		return errors.Wrap(err, "get bucket")
	}

	fi, err := os.Create(localPath)
	if err != nil {
		return errors.Wrap(err, "create file")
	}
	defer fi.Close()

	_, err = cloudprovider.DownloadObjectParallelWithProgress(ctx, bucket, info.Key, nil, fi, 0, 0, false, 10, callback)
	if err != nil {
		return errors.Wrap(err, "download object")
	}

	return nil
}
