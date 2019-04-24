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

package db

import (
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func isNameUnique(manager IModelManager, owner string, name string) (bool, error) {
	q := manager.Query()
	q = manager.FilterByName(q, name)
	if !consts.IsGlobalVirtualResourceNamespace() {
		q = manager.FilterByOwner(q, owner)
	}
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt == 0, nil
}

func NewNameValidator(manager IModelManager, ownerProjId string, name string) error {
	err := manager.ValidateName(name)
	if err != nil {
		return err
	}
	uniq, err := isNameUnique(manager, ownerProjId, name)
	if err != nil {
		return err
	}
	if !uniq {
		return httperrors.NewDuplicateNameError("name", name)
	}
	return nil
}

func isAlterNameUnique(model IModel, name string) (bool, error) {
	manager := model.GetModelManager()
	q := manager.Query()
	q = manager.FilterByName(q, name)
	if !consts.IsGlobalVirtualResourceNamespace() {
		q = manager.FilterByOwner(q, model.GetOwnerProjectId())
	}
	q = manager.FilterByNotId(q, model.GetId())
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt == 0, nil
}

func alterNameValidator(model IModel, name string) error {
	err := model.GetModelManager().ValidateName(name)
	if err != nil {
		return err
	}
	uniq, err := isAlterNameUnique(model, name)
	if err != nil {
		return err
	}
	if !uniq {
		return httperrors.NewDuplicateNameError("name", name)
	}
	return nil
}

func GenerateName(manager IModelManager, ownerProjId string, hint string) (string, error) {
	_, pattern, patternLen := stringutils.ParseNamePattern(hint)
	var name string
	idx := 1
	if patternLen == 0 {
		name = hint
	} else {
		name = fmt.Sprintf(pattern, idx)
		idx += 1
	}
	for {
		uniq, err := isNameUnique(manager, ownerProjId, name)
		if err != nil {
			return "", err
		}
		if uniq {
			return name, nil
		}
		name = fmt.Sprintf(pattern, idx)
		idx += 1
	}
	log.Fatalln("here is not reachable!!!")
	return "", nil
}
