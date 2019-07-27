package multicloud

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SDBInstanceAccountBase struct {
	SResourceBase
}

func (account *SDBInstanceAccountBase) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	return []cloudprovider.ICloudDBInstanceAccountPrivilege{}, nil
}
