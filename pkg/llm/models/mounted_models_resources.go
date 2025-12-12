package models

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	apis "yunion.io/x/onecloud/pkg/apis/llm"
)

type SMountedModelsResourceManager struct {
}

type SMountedModelsResource struct {
	MountedModels []string `charset:"utf8" list:"user" update:"user" create:"optional"`
}

func (manager *SMountedModelsResourceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input apis.MountedModelResourceListInput,
) (*sqlchemy.SQuery, error) {
	if len(input.MountedModels) > 0 {
		q = q.ContainsAny("mounted_models", input.MountedModels)
	}
	return q, nil
}

type MountedModelModelManager interface {
	IsPremountedModelName(fullModelName string) (bool, error)
}

func (manager *SVolumeManager) IsPremountedModelName(fullModelName string) (bool, error) {
	return isPremountedModelName(manager, fullModelName)
}

func (manager *SLLMSkuManager) IsPremountedModelName(fullModelName string) (bool, error) {
	return isPremountedModelName(manager, fullModelName)
}

func isPremountedModelName(manager db.IModelManager, fullModelName string) (bool, error) {
	q := manager.Query().Contains("mounted_models", fmt.Sprintf("%q", fullModelName))
	cnt, err := q.CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "CountWithError")
	}
	return cnt > 0, nil
}
