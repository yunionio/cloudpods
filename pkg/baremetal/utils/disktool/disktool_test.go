package disktool

import (
	"testing"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/pkg/util/stringutils"
)

var (
	term *ssh.Client
)

func init() {
	var err error
	term, err = ssh.NewClient("192.168.0.254", 22, "root", "rMw2qrm6Lb5NVpe0", "")
	if err != nil {
		log.Fatalf("Failed to init ssh client: %v", err)
	}
}

func TestSSHCreate(t *testing.T) {
	tool := NewSSHPartitionTool(term)
	err := tool.FetchDiskConfs([]baremetal.DiskConfiguration{
		{
			Adapter: 0,
			Driver:  baremetal.DISK_DRIVER_LINUX,
		},
	}).RetrieveDiskInfo()
	if err != nil {
		t.Errorf("Failed to RetrieveDiskInfo: %v", err)
	}
	err = tool.RetrievePartitionInfo()
	if err != nil {
		t.Errorf("Failed to RetrievePartitionInfo: %v", err)
	}
	log.Infof("%s", tool.DebugString())

	uuid := stringutils.UUID4
	tool.CreatePartition(-1, 32, "swap", true, baremetal.DISK_DRIVER_LINUX, uuid())
	tool.CreatePartition(-1, 1024, "ext4", true, baremetal.DISK_DRIVER_LINUX, uuid())
	err = tool.CreatePartition(-1, -1, "xfs", true, baremetal.DISK_DRIVER_LINUX, uuid())
	//err = tool.ResizePartition(0, 110*1024)
	if err != nil {
		t.Errorf("Failed to resize fs: %v", err)
	}
}
