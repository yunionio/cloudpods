package models

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func getSyncOwnerProjectId(manager db.IModelManager, userCred mcclient.TokenCredential, projectId string, projectSync bool) string {
	ownerProjId := manager.GetOwnerId(userCred)
	if projectSync && ownerProjId != projectId {
		ownerProjId = projectId
	}
	return ownerProjId
}

type IMetadataSetter interface {
	SetAllMetadata(ctx context.Context, meta map[string]interface{}, userCred mcclient.TokenCredential) error
}

func syncMetadata(ctx context.Context, userCred mcclient.TokenCredential, model IMetadataSetter, remote cloudprovider.ICloudResource) error {
	metaData := remote.GetMetadata()
	if metaData != nil {
		meta := make(map[string]interface{}, 0)
		err := metaData.Unmarshal(meta)
		if err != nil {
			log.Errorf("Get VM Metadata error: %v", err)
			return err
		}

		return model.SetAllMetadata(ctx, meta, userCred)
	}
	return nil
}
