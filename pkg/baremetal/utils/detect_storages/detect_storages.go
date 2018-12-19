package detect_storages

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/baremetal/sysutils"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

func DetectStorageInfo(cli *ssh.Client, wait bool) (interface{}, error) {
	raidDiskInfo := nil

	pcieRet, err := cli.Run("/lib/mos/lsdisk --pcie")
	if err != nil {
		return nil, fmt.Errorf("Fail to retrieve PCIE DISK info")
	}
	pcieDiskInfo := sysutils.ParsePCIEDiskInfo(pcieRet)

	return nil, nil
}
