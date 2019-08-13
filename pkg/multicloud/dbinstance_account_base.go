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

// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
// // Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package multicloud

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SDBInstanceAccountBase struct {
	SResourceBase
}

func (account *SDBInstanceAccountBase) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	return []cloudprovider.ICloudDBInstanceAccountPrivilege{}, nil
}

func (account *SDBInstanceAccountBase) Delete() error {
	return fmt.Errorf("Not Implemented Delete")
}

func (account *SDBInstanceAccountBase) ResetPassword(password string) error {
	return fmt.Errorf("Not Implemented ResetPassword")
}

func (backup *SDBInstanceAccountBase) GrantPrivilege(database, privilege string) error {
	return fmt.Errorf("Not Implement GrantPrivilege")
}

func (backup *SDBInstanceAccountBase) RevokePrivilege(database string) error {
	return fmt.Errorf("Not Implement RevokePrivilege")
}
