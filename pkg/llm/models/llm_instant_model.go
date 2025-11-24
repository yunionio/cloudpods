package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

var llmInstantModelManager *SLLMInstantModelManager

func init() {
	GetLLMInstantModelManager()
}

func GetLLMInstantModelManager() *SLLMInstantModelManager {
	if llmInstantModelManager == nil {
		llmInstantModelManager = &SLLMInstantModelManager{
			SResourceBaseManager: db.NewResourceBaseManager(
				SLLMInstantModel{},
				"llm_instant_models_tbl",
				"llm_with_instant_model",
				"llm_with_instant_models",
			),
		}
		llmInstantModelManager.SetVirtualObject(llmInstantModelManager)
	}
	return llmInstantModelManager
}

type SLLMInstantModelManager struct {
	db.SResourceBaseManager
	db.SStatusResourceBaseManager
}

type SLLMInstantModel struct {
	db.SResourceBase
	db.SStatusResourceBase

	// InstantModelId string `name:"model_id" width:"128" charset:"ascii" nullable:"false" list:"user" primary:"true"`
	LlmId string `width:"128" charset:"ascii" nullable:"false" list:"user" primary:"true"`
	// Model ID, large language model's ID, referring to special model, such as qwen3:8b
	ModelId string `name:"model_id" width:"128" charset:"ascii" nullable:"false" list:"user" primary:"true"`

	// Model Tag
	Tag         string `width:"64" charset:"utf8" nullable:"true" list:"user"`
	DisplayName string `width:"128" charset:"utf8" nullable:"false" list:"user"`
	IsProbed    bool   `list:"user"`
	IsMounted   bool   `list:"user"`
	// IsSystem    tristate.TriState `list:"user"`
}

func (man *SLLMInstantModelManager) fetchLLMInstantModel(llmId string, mdlId string) (*SLLMInstantModel, error) {
	q := man.RawQuery().Equals("llm_id", llmId).Equals("model_id", mdlId)
	llmInstantModel := SLLMInstantModel{}
	err := q.First(&llmInstantModel)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap(err, "Query")
	}
	llmInstantModel.SetModelManager(man, &llmInstantModel)
	return &llmInstantModel, nil
}

func (man *SLLMInstantModelManager) updateInstantModel(ctx context.Context, llmId string, mdlId string, mdlName string, tag string, probed, mounted *bool) (*SLLMInstantModel, error) {
	mdl, err := man.fetchLLMInstantModel(llmId, mdlId)
	if err != nil && errors.Cause(err) != errors.ErrNotFound {
		return nil, errors.Wrap(err, "updateInstantModel")
	}

	mountStr := "nil"
	probedStr := "nil"
	if mounted != nil {
		mountStr = fmt.Sprintf("%v", *mounted)
	}
	if probed != nil {
		probedStr = fmt.Sprintf("%v", *probed)
	}
	log.Debugf("=======updateInstantModel %#v to mounted %s, probed %s", mdl, mountStr, probedStr)

	if mdl == nil {
		// if no such app
		mdl = &SLLMInstantModel{
			LlmId:       llmId,
			ModelId:     mdlId,
			DisplayName: mdlName,
			// IsSystem:    tristate.None,
			// Entry:       entry,
		}
		// if isSys != nil {
		// 	if *isSys {
		// 		mdl.IsSystem = tristate.True
		// 	} else {
		// 		mdl.IsSystem = tristate.False
		// 	}
		// }
		mdl.Tag = tag
		if probed != nil {
			mdl.IsProbed = *probed
		}
		if mounted != nil {
			mdl.IsMounted = *mounted
		}
		mdl.syncStatus()
		err := man.TableSpec().Insert(ctx, mdl)
		if err != nil {
			return nil, errors.Wrap(err, "Insert")
		}
		return mdl, nil
	} else {
		_, err := db.Update(mdl, func() error {
			if len(tag) > 0 {
				mdl.Tag = tag
			}
			if len(mdlName) > 0 {
				mdl.DisplayName = mdlName
			}
			// if isSys != nil {
			// 	if *isSys {
			// 		mdl.IsSystem = tristate.True
			// 	} else {
			// 		mdl.IsSystem = tristate.False
			// 	}
			// }
			// if len(entry) > 0 {
			// 	app.Entry = entry
			// }
			if probed != nil {
				mdl.IsProbed = *probed
				if mdl.IsProbed {
					mdl.Status = api.LLM_STATUS_READY
				} else {
					mdl.Status = api.LLM_STATUS_DELETED
				}
			}

			if mounted != nil {
				mdl.IsMounted = *mounted
			}
			mdl.syncStatus()
			return nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "Update")
		}
	}
	return mdl, nil
}

func (man *SLLMInstantModelManager) filterModels(q *sqlchemy.SQuery, isProbed, isMounted, isSystem *bool) *sqlchemy.SQuery {
	if isProbed != nil {
		if *isProbed {
			q = q.IsTrue("is_probed")
		} else {
			q = q.IsFalse("is_probed")
		}
	}
	if isMounted != nil {
		if *isMounted {
			q = q.IsTrue("is_mounted")
		} else {
			q = q.IsFalse("is_mounted")
		}
	}
	if isSystem != nil {
		if *isSystem {
			q = q.IsTrue("is_system")
		} else {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNull(q.Field("is_system")),
				sqlchemy.IsFalse(q.Field("is_system")),
			))
		}
	}
	return q
}

func (mdl *SLLMInstantModel) FindInstantModel(isInstall bool) (*SInstantModel, error) {
	instMdl, err := GetInstantModelManager().findInstantModel(mdl.ModelId, mdl.Tag, isInstall)
	if err != nil {
		return nil, errors.Wrap(err, "FindInstantModel")
	}
	return instMdl, nil
}

func (mdl *SLLMInstantModel) syncStatus() {
	if !mdl.IsProbed && !mdl.IsMounted {
		mdl.MarkDelete()
	} else {
		mdl.Deleted = false
		mdl.DeletedAt = time.Time{}
	}
}

func (mdl *SLLMInstantModel) getMountPaths(isInstall bool) ([]api.LLMMountDirInfo, error) {
	info, err := mdl.getMountPathsFromImage(isInstall)
	if err != nil {
		return nil, errors.Wrap(err, "getMountPathsFromImage")
	}
	return info, nil
}

func (mdl *SLLMInstantModel) getMountPathsFromImage(isInstall bool) ([]api.LLMMountDirInfo, error) {
	instMdl, err := mdl.FindInstantModel(isInstall)
	if err != nil {
		return nil, errors.Wrap(err, "findInstantApp")
	}
	if instMdl == nil {
		return nil, nil
	}
	info := make([]api.LLMMountDirInfo, 0)
	info = append(info, api.LLMMountDirInfo{
		ImageId: instMdl.ImageId,
	})
	return info, nil
}
