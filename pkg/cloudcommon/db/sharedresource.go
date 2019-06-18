package db

import (
	"context"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"
)

// sharing resoure between project
type SSharedResource struct {
	SResourceBase

	Id int64 `primary:"true" auto_increment:"true" list:"user"`

	ResourceType    string `width:"32" charset:"ascii" nullable:"false" list:"user"`
	ResourceId      string `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
	OwnerProjectId  string `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
	TargetProjectId string `width:"128" charset:"ascii" nullable:"false" index:"true" list:"user"`
}

type SSharedResourceManager struct {
	SResourceBaseManager
}

var SharedResourceManager *SSharedResourceManager

func init() {
	SharedResourceManager = &SSharedResourceManager{
		SResourceBaseManager: NewResourceBaseManager(
			SSharedResource{},
			"shared_resources_tbl",
			"shared_resource",
			"shared_resources",
		),
	}
}

func (manager *SSharedResourceManager) CleanModelSharedProjects(ctx context.Context, userCred mcclient.TokenCredential, model *SVirtualResourceBase) error {
	srs := make([]SSharedResource, 0)
	q := manager.Query()
	err := q.Filter(sqlchemy.AND(
		sqlchemy.Equals(q.Field("owner_project_id"), model.ProjectId),
		sqlchemy.Equals(q.Field("resource_id"), model.GetId()),
		sqlchemy.Equals(q.Field("resource_type"), model.GetModelManager().Keyword()),
	)).All(&srs)
	if err != nil {
		return httperrors.NewInternalServerError("Fetch project error %s", err)
	}
	for i := 0; i < len(srs); i++ {
		srs[i].SetModelManager(manager, &srs[i])
		if err := srs[i].Delete(ctx, userCred); err != nil {
			return httperrors.NewInternalServerError("Unshare project failed %s", err)
		}
	}
	return nil
}
