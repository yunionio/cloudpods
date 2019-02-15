package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/pending_delete"
)

type SImageOptions struct {
	cloudcommon.CommonOptions

	cloudcommon.DBOptions

	pending_delete.SPendingDeleteOptions

	DefaultImageQuota int `default:"10" help:"Common image quota per tenant, default 10"`

	PortV2 int `help:"Listening port for region V2"`

	FilesystemStoreDatadir string `help:"Directory that the Filesystem backend store writes image data to"`

	TorrentStoreDir string `help:"directory to store image torrent files"`

	EnableTorrentService bool `help:"Enable torrent service" default:"false"`

	TargetImageFormats []string `help:"target image formats that the system will automatically convert to" default:"qcow2,vmdk,vhd"`

	TorrentClientPath string `help:"path to torrent executable" default:"/opt/yunion/bin/torrent"`
}

var (
	Options SImageOptions
)
