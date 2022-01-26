package db

import (
	"context"

	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	dbapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

type SSecretResourceBaseModelManager struct{}

func (secret *SSecretResourceBaseModelManager) ListItemFilter(manager IModelManager, q *sqlchemy.SQuery, input apis.SecretResourceListInput) *sqlchemy.SQuery {
	if input.SecretLevel == "" {
		return q
	}
	inputTagFilters := tagutils.STagFilters{}
	tag := tagutils.STag{
		Key:   CLASS_TAG_PREFIX + dbapi.SECRET_KEY,
		Value: input.SecretLevel,
	}
	var tagSet tagutils.TTagSet = []tagutils.STag{tag}
	inputTagFilters.AddFilter(tagSet)
	return ObjectIdQueryWithTagFilters(q, "id", manager.Keyword(), inputTagFilters)
}

func (secret *SSecretResourceBaseModelManager) FetchCustomizeColumns(manager IModelManager, userCred mcclient.TokenCredential, objs []interface{}, fields stringutils2.SSortedStrings) []apis.SecretResourceInfo {
	ret := make([]apis.SecretResourceInfo, len(objs))
	resIds := make([]string, len(objs))
	for i := range objs {
		resIds[i] = GetModelIdstr(objs[i].(IModel))
	}
	if fields == nil || fields.Contains("secret_level") {
		q := Metadata.Query("id", "key", "value").Equals("key", CLASS_TAG_PREFIX+dbapi.SECRET_KEY)
		metaKeyValues := make(map[string][]SMetadata)
		err := FetchQueryObjectsByIds(q, "id", resIds, &metaKeyValues)
		if err != nil {
			log.Errorf("FetchQueryObjectsByIds metadata fail %s", err)
			return ret
		}
		for i := range objs {
			if metaList, ok := metaKeyValues[resIds[i]]; ok {
				ret[i].SecretLevel = metaList[0].Value
			}
		}
	}
	return ret
}

func (secret *SSecretResourceBaseModelManager) SetSecretLevel(ctx context.Context, userCred mcclient.TokenCredential, model *SStandaloneAnonResourceBase, secretLevel string) error {
	return model.SetClassMetadataAll(ctx, map[string]interface{}{
		dbapi.SECRET_KEY: secretLevel,
	}, userCred)
}

func (secret *SSecretResourceBaseModelManager) RemoveSecretLevel(ctx context.Context, userCred mcclient.TokenCredential, model *SStandaloneAnonResourceBase) error {
	return model.RemoveMetadata(ctx, CLASS_TAG_PREFIX+dbapi.SECRET_KEY, userCred)
}
