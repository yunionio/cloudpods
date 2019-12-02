package quotas

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

const (
	QuotaLockName = "quotas"
)

func LockQuota(ctx context.Context, quota IQuota) {
	LockQuotaKeys(ctx, quota.GetKeys())
}

func ReleaseQuota(ctx context.Context, quota IQuota) {
	ReleaseQuotaKeys(ctx, quota.GetKeys())
}

func LockQuotaKeys(ctx context.Context, keys IQuotaKeys) {
	lockman.LockRawObject(ctx, QuotaLockName, QuotaKeyString(keys))
}

func ReleaseQuotaKeys(ctx context.Context, keys IQuotaKeys) {
	lockman.ReleaseRawObject(ctx, QuotaLockName, QuotaKeyString(keys))
}
