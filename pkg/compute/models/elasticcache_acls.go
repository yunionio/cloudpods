package models

import (
	"context"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// SElasticcache.Acl
type SElasticcacheAclManager struct {
	db.SStandaloneResourceBaseManager
}

var ElasticcacheAclManager *SElasticcacheAclManager

func init() {
	ElasticcacheAclManager = &SElasticcacheAclManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SElasticcacheAcl{},
			"elasticcacheacls_tbl",
			"elasticcacheacl",
			"elasticcacheacls",
		),
	}
	ElasticcacheAclManager.SetVirtualObject(ElasticcacheAclManager)
}

type SElasticcacheAcl struct {
	db.SStandaloneResourceBase
	db.SExternalizedResourceBase

	ElasticcacheId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"` // elastic cache instance id

	IpList string `width:"256" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
}

func (manager *SElasticcacheAclManager) SyncElasticcacheAcls(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, cloudElasticcacheAcls []cloudprovider.ICloudElasticcacheAcl) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, elasticcache.GetOwnerId()))

	syncResult := compare.SyncResult{}

	dbAcls, err := elasticcache.GetElasticcacheAcls()
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SElasticcacheAcl, 0)
	commondb := make([]SElasticcacheAcl, 0)
	commonext := make([]cloudprovider.ICloudElasticcacheAcl, 0)
	added := make([]cloudprovider.ICloudElasticcacheAcl, 0)
	if err := compare.CompareSets(dbAcls, cloudElasticcacheAcls, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudElasticcacheAcl(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudElasticcacheAcl(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}

		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		_, err := manager.newFromCloudElasticcacheAcl(ctx, userCred, elasticcache, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}

		syncResult.Add()
	}
	return syncResult
}

func (self *SElasticcacheAcl) syncRemoveCloudElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return errors.Wrapf(err, "newFromCloudElasticcacheAcl.Remove")
	}
	return self.Delete(ctx, userCred)
}

func (self *SElasticcacheAcl) SyncWithCloudElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, extAcl cloudprovider.ICloudElasticcacheAcl) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		self.IpList = extAcl.GetIpList()

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "SyncWithCloudElasticcacheAcl.UpdateWithLock")
	}

	return nil
}

func (manager *SElasticcacheAclManager) newFromCloudElasticcacheAcl(ctx context.Context, userCred mcclient.TokenCredential, elasticcache *SElasticcache, extAcl cloudprovider.ICloudElasticcacheAcl) (*SElasticcacheAcl, error) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	acl := SElasticcacheAcl{}
	acl.SetModelManager(manager, &acl)

	acl.ElasticcacheId = elasticcache.GetId()
	acl.Name = extAcl.GetName()
	acl.ExternalId = extAcl.GetGlobalId()
	acl.IpList = extAcl.GetIpList()

	err := manager.TableSpec().Insert(&acl)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudElasticcacheAcl.Insert")
	}

	return &acl, nil
}
