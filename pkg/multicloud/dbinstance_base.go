package multicloud

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SDBInstanceBase struct {
	SVirtualResourceBase
}

func (instance *SDBInstanceBase) GetConnectionStr() string {
	return ""
}

func (instance *SDBInstanceBase) GetInternalConnectionStr() string {
	return ""
}

func (instance *SDBInstanceBase) GetDBNetwork() (*cloudprovider.SDBInstanceNetwork, error) {
	return nil, fmt.Errorf("Not Implemented GetDBNetwork")
}

func (instance *SDBInstanceBase) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	return nil, fmt.Errorf("Not Implemented GetIDBInstanceParameters")
}

func (instance *SDBInstanceBase) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	return nil, fmt.Errorf("Not Implemented GetIDBInstanceDatabases")
}

func (instance *SDBInstanceBase) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	return nil, fmt.Errorf("Not Implemented GetIDBInstanceAccounts")
}

func (instance *SDBInstanceBase) GetCategory() string {
	return ""
}
