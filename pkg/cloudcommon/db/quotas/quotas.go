package quotas

import (
	"context"
	"reflect"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db/lockman"
)

const (
	METADATA_KEY = "quota"
)

type IQuota interface {
	FetchSystemQuota()
	FetchUsage(projectId string) error
	Update(quota IQuota)
	Add(quota IQuota)
	Sub(quota IQuota)
	Exceed(quota IQuota) error
	IsEmpty() bool
	ToJSON(prefix string) jsonutils.JSONObject
}

type SQuotaManager struct {
	keyword        string
	quotaType      reflect.Type
	persistenStore IQuotaStore
	pendingStore   IQuotaStore
}

func (manager *SQuotaManager) Keyword() string {
	return manager.keyword
}

func NewQuotaManager(keyword string, quotaData interface{}, persist IQuotaStore, pending IQuotaStore) *SQuotaManager {
	quotaType := reflect.Indirect(reflect.ValueOf(quotaData)).Type()
	man := SQuotaManager{keyword: keyword, quotaType: quotaType, persistenStore: persist, pendingStore: pending}
	return &man
}

func (manager *SQuotaManager) newQuota() IQuota {
	val := reflect.New(manager.quotaType)
	return val.Interface().(IQuota)
}

func (manager *SQuotaManager) CancelPendingUsage(ctx context.Context, userCred mcclient.TokenCredential, projectId string, localUsage IQuota, cancelUsage IQuota) error {
	lockman.LockClass(ctx, manager, projectId)
	defer lockman.ReleaseClass(ctx, manager, projectId)

	return manager._cancelPendingUsage(ctx, userCred, projectId, localUsage, cancelUsage)
}

func (manager *SQuotaManager) _cancelPendingUsage(ctx context.Context, userCred mcclient.TokenCredential, projectId string, localUsage IQuota, cancelUsage IQuota) error {

	quota := manager.newQuota()
	err := manager.pendingStore.GetQuota(ctx, projectId, quota)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	localUsage.Sub(cancelUsage)
	quota.Sub(cancelUsage)
	err = manager.pendingStore.SetQuota(ctx, userCred, projectId, quota)
	if err != nil {
		log.Errorf("%s", err)
	}
	return err
}

func (manager *SQuotaManager) GetPendingUsage(ctx context.Context, projectId string, quota IQuota) error {
	return manager.pendingStore.GetQuota(ctx, projectId, quota)
}

func (manager *SQuotaManager) GetQuota(ctx context.Context, projectId string, quota IQuota) error {
	err := manager.persistenStore.GetQuota(ctx, projectId, quota)
	if err != nil {
		return err
	}
	if quota.IsEmpty() {
		quota.FetchSystemQuota()
	}
	return nil
}

func (manager *SQuotaManager) SetQuota(ctx context.Context, userCred mcclient.TokenCredential, projectId string, quota IQuota) error {
	lockman.LockClass(ctx, manager, projectId)
	defer lockman.ReleaseClass(ctx, manager, projectId)

	return manager._setQuota(ctx, userCred, projectId, quota)
}

func (manager *SQuotaManager) _setQuota(ctx context.Context, userCred mcclient.TokenCredential, projectId string, quota IQuota) error {

	return manager.persistenStore.SetQuota(ctx, userCred, projectId, quota)
}

func (manager *SQuotaManager) CheckQuota(ctx context.Context, userCred mcclient.TokenCredential, projectId string, quota IQuota) (IQuota, error) {
	lockman.LockClass(ctx, manager, projectId)
	defer lockman.ReleaseClass(ctx, manager, projectId)

	return manager._checkQuota(ctx, userCred, projectId, quota)
}

func (manager *SQuotaManager) _checkQuota(ctx context.Context, userCred mcclient.TokenCredential, projectId string, quota IQuota) (IQuota, error) {
	stored := manager.newQuota()
	err := manager.GetQuota(ctx, projectId, stored)
	if err != nil {
		log.Errorf("fail to get quota %s", err)
		return nil, err
	}
	used := manager.newQuota()
	err = used.FetchUsage(projectId)
	if err != nil {
		log.Errorf("fail to get quota usage %s", err)
		return nil, err
	}

	pending := manager.newQuota()
	err = manager.GetPendingUsage(ctx, projectId, pending)
	if err != nil {
		log.Errorf("fail to get pending usage %s", err)
		return nil, err
	}

	used.Add(pending)
	used.Add(quota)

	err = used.Exceed(stored)
	if err != nil {
		return nil, err
	}

	return used, nil
}

func (manager *SQuotaManager) CheckSetPendingQuota(ctx context.Context, userCred mcclient.TokenCredential, projectId string, quota IQuota) error {
	lockman.LockClass(ctx, manager, projectId)
	defer lockman.ReleaseClass(ctx, manager, projectId)

	return manager._checkSetPendingQuota(ctx, userCred, projectId, quota)
}

func (manager *SQuotaManager) _checkSetPendingQuota(ctx context.Context, userCred mcclient.TokenCredential, projectId string, quota IQuota) error {
	_, err := manager._checkQuota(ctx, userCred, projectId, quota)
	if err != nil {
		return err
	}
	pending := manager.newQuota()
	err = manager.pendingStore.GetQuota(ctx, projectId, pending)
	if err != nil {
		log.Errorf("GetQuota fail %s", err)
		return err
	}
	pending.Add(quota)
	return manager.pendingStore.SetQuota(ctx, userCred, projectId, pending)
}
