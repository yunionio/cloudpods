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
	"regexp"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func isNameUnique(manager IModelManager, ownerId mcclient.IIdentityProvider, name string, uniqValues jsonutils.JSONObject) (bool, error) {
	return isRawNameUnique(manager, ownerId, name, uniqValues, false)
}

func isRawNameUnique(manager IModelManager, ownerId mcclient.IIdentityProvider, name string, uniqValues jsonutils.JSONObject, isRaw bool) (bool, error) {
	var q *sqlchemy.SQuery
	if isRaw {
		q = manager.TableSpec().Instance().Query()
	} else {
		q = manager.Query()
	}
	q = manager.FilterByName(q, name)
	q = manager.FilterByOwner(q, ownerId, manager.NamespaceScope())
	if !isRaw {
		q = manager.FilterBySystemAttributes(q, nil, nil, manager.ResourceScope())
		if uniqValues != nil {
			q = manager.FilterByUniqValues(q, uniqValues)
		}
	}
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt == 0, nil
}

func NewNameValidator(manager IModelManager, ownerId mcclient.IIdentityProvider, name string, uniqValues jsonutils.JSONObject) error {
	err := manager.ValidateName(name)
	if err != nil {
		return err
	}
	uniq, err := isNameUnique(manager, ownerId, name, uniqValues)
	if err != nil {
		return err
	}
	if !uniq {
		return httperrors.NewDuplicateNameError(manager.Keyword(), name)
	}
	return nil
}

func isAlterNameUnique(model IModel, name string) (bool, error) {
	return isRawAlterNameUnique(model, name, false)
}

func isRawAlterNameUnique(model IModel, name string, isRaw bool) (bool, error) {
	manager := model.GetModelManager()
	var q *sqlchemy.SQuery
	if isRaw {
		q = manager.TableSpec().Instance().Query()
	} else {
		q = manager.Query()
	}
	q = manager.FilterByName(q, name)
	q = manager.FilterByOwner(q, model.GetOwnerId(), manager.NamespaceScope())
	q = manager.FilterByNotId(q, model.GetId())
	if !isRaw {
		q = manager.FilterBySystemAttributes(q, nil, nil, manager.ResourceScope())
		if uniqValues := model.GetUniqValues(); uniqValues != nil {
			q = manager.FilterByUniqValues(q, uniqValues)
		}
	}
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

func GenerateName(manager IModelManager, ownerId mcclient.IIdentityProvider, hint string) (string, error) {
	return GenerateName2(manager, ownerId, hint, nil, 1)
}

func GenerateAlterName(model IModel, hint string) (string, error) {
	if hint == model.GetName() {
		return hint, nil
	}
	return GenerateName2(nil, nil, hint, model, 1)
}

func GenerateName2(manager IModelManager, ownerId mcclient.IIdentityProvider, hint string, model IModel, baseIndex int) (string, error) {
	_, pattern, patternLen, offset := stringutils2.ParseNamePattern2(hint)
	var name string
	if patternLen == 0 {
		name = hint
	} else {
		if offset > 0 {
			baseIndex = offset
		}
		name = fmt.Sprintf(pattern, baseIndex)
		baseIndex += 1
	}
	for {
		var uniq bool
		var err error
		if model == nil {
			uniq, err = isRawNameUnique(manager, ownerId, name, nil, consts.IsHistoricalUniqueName())
		} else {
			uniq, err = isRawAlterNameUnique(model, name, consts.IsHistoricalUniqueName())
		}
		if err != nil {
			return "", err
		}
		if uniq {
			return name, nil
		}
		name = fmt.Sprintf(pattern, baseIndex)
		baseIndex += 1
	}
}

var (
	dnsNameREG = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

type SDnsNameValidatorManager struct{}

func (manager *SDnsNameValidatorManager) ValidateName(name string) error {
	if dnsNameREG.MatchString(name) {
		return nil
	}
	return httperrors.NewInputParameterError("name starts with letter, and contains letter, number and - only")
}

var (
	hostNameREG = regexp.MustCompile(`^[a-z$][a-z0-9-${}.]*$`)
)

type SHostNameValidatorManager struct{}

func (manager *SHostNameValidatorManager) ValidateName(name string) error {
	if hostNameREG.MatchString(name) {
		return nil
	}
	return httperrors.NewInputParameterError("name starts with letter, and contains letter, number and - only")
}
