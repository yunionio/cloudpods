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

package models

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type IEipAssociateInstance interface {
	db.IStatusStandaloneModel
	GetVpc() (*SVpc, error)
}

func ValidateAssociateEip(obj IEipAssociateInstance) error {
	vpc, err := obj.GetVpc()
	if err != nil {
		return httperrors.NewGeneralError(err)
	}

	if vpc != nil {
		if !vpc.IsSupportAssociateEip() {
			return httperrors.NewNotSupportedError("resource %s in vpc %s external access mode %s is not support accociate eip", obj.GetName(), vpc.GetName(), vpc.ExternalAccessMode)
		}
	}

	return nil
}
