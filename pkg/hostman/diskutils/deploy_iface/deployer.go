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

package deploy_iface

import (
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type IDeployer interface {
	Connect(desc *apis.GuestDesc) error
	Disconnect() error

	GetPartitions() []fsdriver.IDiskPartition
	IsLVMPartition() bool
	Zerofree()
	ResizePartition() error
	FormatPartition(fs, uuid string, features *apis.FsFeatures) error
	MakePartition(fs string) error

	MountRootfs(readonly bool) (fsdriver.IRootFsDriver, error)
	UmountRootfs(fd fsdriver.IRootFsDriver) error
	DetectIsUEFISupport(rootfs fsdriver.IRootFsDriver) bool

	DeployGuestfs(req *apis.DeployParams) (res *apis.DeployGuestFsResponse, err error)
	ResizeFs() (res *apis.Empty, err error)
	FormatFs(req *apis.FormatFsParams) (*apis.Empty, error)
	SaveToGlance(req *apis.SaveToGlanceParams) (*apis.SaveToGlanceResponse, error)
	ProbeImageInfo(req *apis.ProbeImageInfoPramas) (*apis.ImageInfo, error)
}
