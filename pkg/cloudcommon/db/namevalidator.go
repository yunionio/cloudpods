package db

import (
	"fmt"

	"github.com/yunionio/pkg/httperrors"
	"github.com/yunionio/pkg/util/stringutils"
)

func isNameUnique(manager IModelManager, ownerProjId string, name string) bool {
	q := manager.Query()
	q = manager.FilterByName(q, name)
	q = manager.FilterByOwner(q, ownerProjId)
	return q.Count() == 0
}

func newNameValidator(manager IModelManager, ownerProjId string, name string) error {
	err := manager.ValidateName(name)
	if err != nil {
		return err
	}
	if !isNameUnique(manager, ownerProjId, name) {
		return httperrors.NewConflictError(fmt.Sprintf("duplicate name %s", name))
	}
	return nil
}

func isAlterNameUnique(model IModel, name string) bool {
	manager := model.GetModelManager()
	q := manager.Query()
	q = manager.FilterByName(q, name)
	q = manager.FilterByOwner(q, model.GetOwnerProjectId())
	q = manager.FilterByNotId(q, model.GetId())
	return q.Count() == 0
}

func alterNameValidator(model IModel, name string) error {
	err := model.GetModelManager().ValidateName(name)
	if err != nil {
		return err
	}
	if !isAlterNameUnique(model, name) {
		return httperrors.NewConflictError(fmt.Sprintf("duplicate name %s", name))
	}
	return nil
}

func GenerateName(manager IModelManager, ownerProjId string, hint string) string {
	_, pattern, patternLen := stringutils.ParseNamePattern(hint)
	var name string
	idx := 1
	if patternLen == 0 {
		name = hint
	} else {
		name = fmt.Sprintf(pattern, idx)
		idx += 1
	}
	for !isNameUnique(manager, ownerProjId, name) {
		name = fmt.Sprintf(pattern, idx)
		idx += 1
	}
	return name
}
