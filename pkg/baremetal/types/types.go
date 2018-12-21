package types

import (
	"net"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/baremetal/utils/disktool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type IBaremetalServer interface {
	GetName() string
	GetId() string
	RemoveDesc()
	DoDiskUnconfig(term *ssh.Client) error
	DoDiskConfig(term *ssh.Client) error
	DoEraseDisk(term *ssh.Client) error
	DoPartitionDisk(term *ssh.Client) ([]*disktool.Partition, error)
	DoRebuildRootDisk(term *ssh.Client) ([]*disktool.Partition, error)
	SyncPartitionSize(term *ssh.Client, parts []*disktool.Partition) ([]jsonutils.JSONObject, error)
	DoDeploy(term *ssh.Client, data jsonutils.JSONObject, isInit bool) (jsonutils.JSONObject, error)
	SaveDesc(desc jsonutils.JSONObject) error
	GetNicByMac(mac net.HardwareAddr) *types.Nic
}
