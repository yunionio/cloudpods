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
