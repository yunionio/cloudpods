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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SDBInstanceAccountBase struct {
}

func (account *SDBInstanceAccountBase) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	return []cloudprovider.ICloudDBInstanceAccountPrivilege{}, nil
}

func (account *SDBInstanceAccountBase) Delete() error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "Delete")
}

func (account *SDBInstanceAccountBase) GetHost() string {
	return "%"
}

func (account *SDBInstanceAccountBase) GetStatus() string {
	return api.DBINSTANCE_USER_AVAILABLE
}

func (account *SDBInstanceAccountBase) ResetPassword(password string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "ResetPassword")
}

func (backup *SDBInstanceAccountBase) GrantPrivilege(database, privilege string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "GrantPrivilege")
}

func (backup *SDBInstanceAccountBase) RevokePrivilege(database string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RevokePrivilege")
}
