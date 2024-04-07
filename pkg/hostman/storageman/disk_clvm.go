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

package storageman

import (
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SCLVMDisk struct {
	SLVMDisk
}

func NewCLVMDisk(storage IStorage, id string) *SCLVMDisk {
	return &SCLVMDisk{
		SLVMDisk: *NewLVMDisk(storage, id),
	}
}

func (d *SCLVMDisk) PrepareMigrate(liveMigrate bool) ([]string, string, bool, error) {
	return nil, "", false, fmt.Errorf("Not support")
}

func (d *SCLVMDisk) CreateFromImageFuse(ctx context.Context, url string, size int64, encryptInfo *apis.SEncryptInfo) error {
	return fmt.Errorf("Not support")
}

func (d *SCLVMDisk) GetType() string {
	return api.STORAGE_CLVM
}
