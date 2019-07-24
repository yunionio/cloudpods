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

package multicloud

import "yunion.io/x/onecloud/pkg/cloudprovider"

type SNoObjectStorageRegion struct{}

///////////////// S3 ///////////////////

func (cli *SNoObjectStorageRegion) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SNoObjectStorageRegion) CreateIBucket(name string, storageClassStr string, aclStr string) error {
	return cloudprovider.ErrNotSupported
}

func (cli *SNoObjectStorageRegion) DeleteIBucket(name string) error {
	return cloudprovider.ErrNotSupported
}

func (cli *SNoObjectStorageRegion) IBucketExist(name string) (bool, error) {
	return false, cloudprovider.ErrNotSupported
}

func (cli *SNoObjectStorageRegion) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return nil, cloudprovider.ErrNotSupported
}

////////////////// END S3 fake API //////////
