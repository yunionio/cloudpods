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
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDeletePreventableResourceBase struct {
	DisableDelete tristate.TriState `nullable:"false" default:"true" list:"user" update:"user" create:"optional"`
}

func (lock *SDeletePreventableResourceBase) MarkDeletePreventionOff() {
	lock.DisableDelete = tristate.False
}

func (lock *SDeletePreventableResourceBase) MarkDeletePreventionOn() {
	lock.DisableDelete = tristate.True
}

func (lock *SDeletePreventableResourceBase) DeletePreventionOn(model db.IModel, userCred mcclient.TokenCredential) error {
	_, err := db.Update(model, func() error {
		model.MarkDeletePreventionOn()
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (lock *SDeletePreventableResourceBase) DeletePreventionOff(model db.IModel, userCred mcclient.TokenCredential) error {
	_, err := db.Update(model, func() error {
		model.MarkDeletePreventionOff()
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
