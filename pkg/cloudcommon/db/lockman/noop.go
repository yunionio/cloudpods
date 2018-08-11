package lockman

import (
	"context"

	"yunion.io/x/log"
)

type SNoopLockManager struct {
}

func (lockman *SNoopLockManager) LockKey(ctx context.Context, key string) {
	log.Debugf("LockKey %s in context %#v", key, ctx)
}

func (lockman *SNoopLockManager) UnlockKey(ctx context.Context, key string) {
	log.Debugf("UnlockKey %s in context %#v", key, ctx)
}

func NewNoopLockManager() ILockManager {
	lockMan := SNoopLockManager{}
	return &lockMan
}
