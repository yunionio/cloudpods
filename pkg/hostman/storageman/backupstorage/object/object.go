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

package object

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/streamutils"
)

type SObjectBackupStorage struct {
	BackupStorageId string

	bucket string

	store *objectstore.SObjectStoreClient
}

func newObjectBackupStorage(backupStorageId, bucketUrl, accessKey, secret string, signVer objectstore.S3SignVersion) (*SObjectBackupStorage, error) {
	bucket, endpoint, err := parseBucketUrl(bucketUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "parseBucketUrl %s", bucketUrl)
	}
	cfg := objectstore.NewObjectStoreClientConfig(endpoint, accessKey, secret)
	if len(signVer) > 0 {
		cfg = cfg.SignVersion(signVer)
	}
	store, err := objectstore.NewObjectStoreClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "NewObjectStoreClient")
	}

	return &SObjectBackupStorage{
		BackupStorageId: backupStorageId,

		bucket: bucket,

		store: store,
	}, nil
}

func parseBucketUrl(bucketUrl string) (string, string, error) {
	bu, err := url.Parse(bucketUrl)
	if err != nil {
		return "", "", errors.Wrapf(err, "ur.Parse %s", bucketUrl)
	}
	for len(bu.Path) > 0 && bu.Path[0] == '/' {
		bu.Path = bu.Path[1:]
	}
	if len(bu.Path) > 0 {
		bucket := strings.TrimRight(bu.Path, "/")
		bu.Path = ""
		return bucket, fmt.Sprintf("%s://%s", bu.Scheme, bu.Host), nil
	} else {
		parts := strings.Split(bu.Host, ".")
		if len(parts) < 3 {
			return "", "", errors.Wrapf(errors.ErrInvalidFormat, "host %s should have at least 3 segments", bu.Host)
		}
		return parts[0], fmt.Sprintf("%s//%s", bu.Scheme, bu.Host), nil
	}
}

const backupPathPrefix = "backups"
const backupInstancePathPrefix = "backuppacks"

func (s *SObjectBackupStorage) getBackupKey(backupId string) string {
	return fmt.Sprintf("%s/%s", backupPathPrefix, backupId)
}

func (s *SObjectBackupStorage) getBackupInstanceKey(backupInstancePackName string) string {
	return fmt.Sprintf("%s/%s", backupInstancePathPrefix, backupInstancePackName)
}

func (s *SObjectBackupStorage) getBucket() (cloudprovider.ICloudBucket, error) {
	bucket, err := s.store.GetIRegion().GetIBucketByName(s.bucket)
	if err != nil {
		return nil, errors.Wrap(err, "IBucketExist")
	}
	return bucket, nil
}

func (s *SObjectBackupStorage) SaveBackupFrom(ctx context.Context, srcFilename string, backupId string) error {
	return s.saveObject(ctx, srcFilename, backupId, s.getBackupKey)
}

func (s *SObjectBackupStorage) SaveBackupInstanceFrom(ctx context.Context, srcFilename string, backupId string) error {
	return s.saveObject(ctx, srcFilename, backupId, s.getBackupInstanceKey)
}

func (s *SObjectBackupStorage) saveObject(ctx context.Context, srcFilename string, id string, getKeyFunc func(string) string) error {
	bucket, err := s.getBucket()
	if err != nil {
		return errors.Wrap(err, "getBucket")
	}
	fileInfo, err := os.Stat(srcFilename)
	if err != nil {
		return errors.Wrapf(err, "stat %s", srcFilename)
	}
	file, err := os.Open(srcFilename)
	if err != nil {
		return errors.Wrapf(err, "Open %s", srcFilename)
	}
	defer file.Close()

	err = cloudprovider.UploadObject(ctx, bucket, getKeyFunc(id), 200*1024*1024, file, fileInfo.Size(), cloudprovider.ACLPrivate, "", nil, false)
	if err != nil {
		return errors.Wrapf(err, "UploadObject %s %s", srcFilename, getKeyFunc(id))
	}

	return nil
}

func (s *SObjectBackupStorage) RestoreBackupTo(ctx context.Context, targetFilename string, backupId string) error {
	return s.restoreObject(ctx, targetFilename, backupId, s.getBackupKey)
}

func (s *SObjectBackupStorage) RestoreBackupInstanceTo(ctx context.Context, targetFilename string, backupId string) error {
	return s.restoreObject(ctx, targetFilename, backupId, s.getBackupInstanceKey)
}

func (s *SObjectBackupStorage) restoreObject(ctx context.Context, targetFilename string, id string, getKeyFunc func(string) string) error {
	bucket, err := s.getBucket()
	if err != nil {
		return errors.Wrap(err, "getBucket")
	}
	reader, err := bucket.GetObject(ctx, getKeyFunc(id), nil)
	if err != nil {
		return errors.Wrap(err, "GetObject")
	}
	file, err := os.OpenFile(targetFilename, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return errors.Wrapf(err, "OpenFile %s", targetFilename)
	}
	defer file.Close()
	_, err = streamutils.StreamPipe(reader, file, false, nil)
	if err != nil {
		return errors.Wrap(err, "StreamPipe")
	}
	return nil
}

func (s *SObjectBackupStorage) RemoveBackup(ctx context.Context, backupId string) error {
	return s.removeObject(ctx, backupId, s.getBackupKey)
}

func (s *SObjectBackupStorage) RemoveBackupInstance(ctx context.Context, backupId string) error {
	return s.removeObject(ctx, backupId, s.getBackupInstanceKey)
}

func (s *SObjectBackupStorage) removeObject(ctx context.Context, id string, getKeyFunc func(string) string) error {
	bucket, err := s.getBucket()
	if err != nil {
		return errors.Wrap(err, "getBucket")
	}
	err = bucket.DeleteObject(ctx, getKeyFunc(id))
	if err != nil {
		return errors.Wrap(err, "DeleteObject")
	}
	return nil
}

func (s *SObjectBackupStorage) IsBackupExists(backupId string) (bool, error) {
	return s.isObjectExists(backupId, s.getBackupKey)
}

func (s *SObjectBackupStorage) IsBackupInstanceExists(backupId string) (bool, error) {
	return s.isObjectExists(backupId, s.getBackupInstanceKey)
}

func (s *SObjectBackupStorage) isObjectExists(id string, getKeyFunc func(string) string) (bool, error) {
	bucket, err := s.getBucket()
	if err != nil {
		return false, errors.Wrap(err, "getBucket")
	}
	_, err = cloudprovider.GetIObject(bucket, getKeyFunc(id))
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return false, nil
		}
		return false, errors.Wrap(err, "GetIObject")
	}
	return true, nil
}

func (s *SObjectBackupStorage) IsOnline() (bool, string, error) {
	exist, err := s.store.GetIRegion().IBucketExist(s.bucket)
	if err != nil {
		return false, "", errors.Wrap(err, "IBucketExist")
	}
	return exist, "", nil
}
