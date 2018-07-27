package lockman

import (
	"context"
	"fmt"
)

type ILockedClass interface {
	Keyword() string
}

type ILockedObject interface {
	ILockedClass
	GetId() string
}

type ILockManager interface {
	LockKey(ctx context.Context, key string)
	UnlockKey(ctx context.Context, key string)
}

func getClassKey(manager ILockedClass, projectId string) string {
	// assert(getattr(cls, '_resource_name_', None) is not None)
	// return '%s-%s' % (cls._resource_name_, user_cred.tenant_id)
	return fmt.Sprintf("%s-%s", manager.Keyword(), projectId)
}

func getObjectKey(model ILockedObject) string {
	// assert(getattr(obj, '_resource_name_', None) is not None)
	// assert(getattr(obj, 'id', None) is not None)
	// return '%s-%s' % (obj._resource_name_, obj.id)
	return getRawObjectKey(model.Keyword(), model.GetId())
}

func getRawObjectKey(resName string, resId string) string {
	return fmt.Sprintf("%s-%s", resName, resId)
}

func getJointObjectKey(model ILockedObject, model2 ILockedObject) string {
	// def _get_joint_object_key(self, obj1, obj2, user_cred):
	// return '%s-%s' % (self._get_object_key(obj1, user_cred),
	//		self._get_object_key(obj2, user_cred))
	return fmt.Sprintf("%s-%s", getObjectKey(model), getObjectKey(model2))
}

var _lockman ILockManager

func Init(man ILockManager) {
	_lockman = man
}

func LockClass(ctx context.Context, manager ILockedClass, projectId string) {
	key := getClassKey(manager, projectId)
	_lockman.LockKey(ctx, key)
}

func ReleaseClass(ctx context.Context, manager ILockedClass, projectId string) {
	key := getClassKey(manager, projectId)
	_lockman.UnlockKey(ctx, key)
}

func LockObject(ctx context.Context, model ILockedObject) {
	key := getObjectKey(model)
	_lockman.LockKey(ctx, key)
}

func ReleaseObject(ctx context.Context, model ILockedObject) {
	key := getObjectKey(model)
	_lockman.UnlockKey(ctx, key)
}

func LockRawObject(ctx context.Context, resName string, resId string) {
	key := getRawObjectKey(resName, resId)
	_lockman.LockKey(ctx, key)
}

func ReleaseRawObject(ctx context.Context, resName string, resId string) {
	key := getRawObjectKey(resName, resId)
	_lockman.UnlockKey(ctx, key)
}

func LockJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject) {
	key := getJointObjectKey(model, model2)
	_lockman.LockKey(ctx, key)
}

func ReleaseJointObject(ctx context.Context, model ILockedObject, model2 ILockedObject) {
	key := getJointObjectKey(model, model2)
	_lockman.UnlockKey(ctx, key)
}
