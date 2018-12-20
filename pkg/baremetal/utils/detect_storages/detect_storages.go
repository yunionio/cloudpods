package detect_storages

import (
	//"fmt"

	"yunion.io/x/onecloud/pkg/util/ssh"
	//"yunion.io/x/onecloud/pkg/util/sysutils"
)

func DetectStorageInfo(cli *ssh.Client, wait bool) (interface{}, error) {
	//var raidDiskInfo interface{}

	//pcieRet, err := cli.Run("/lib/mos/lsdisk --pcie")
	//if err != nil {
	//return nil, fmt.Errorf("Fail to retrieve PCIE DISK info")
	//}
	//pcieDiskInfo := sysutils.ParsePCIEDiskInfo(pcieRet)

	return nil, nil
}
