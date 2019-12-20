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

package options

import (
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/pending_delete"
)

type SImageOptions struct {
	common_options.CommonOptions

	common_options.DBOptions

	pending_delete.SPendingDeleteOptions

	DefaultImageQuota int `default:"10" help:"Common image quota per tenant, default 10"`

	PortV2 int `help:"Listening port for region V2"`

	FilesystemStoreDatadir string `help:"Directory that the Filesystem backend store writes image data to"`

	TorrentStoreDir string `help:"directory to store image torrent files"`

	EnableTorrentService bool `help:"Enable torrent service" default:"false"`

	TargetImageFormats []string `help:"target image formats that the system will automatically convert to" default:"qcow2,vmdk,vhd"`

	TorrentClientPath string `help:"path to torrent executable" default:"/opt/yunion/bin/torrent"`

	DeployServerSocketPath string `help:"Deploy server listen socket path" default:"/var/run/deploy.sock"`
}

var (
	Options SImageOptions
)

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*SImageOptions)
	newOpts := newO.(*SImageOptions)

	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		return true
	}

	return false
}
