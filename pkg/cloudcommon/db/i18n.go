package db

import (
	"context"

	"github.com/pkg/errors"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IModelI18nBase interface {
	GetKeyValue() string
	Lookup(tag i18n.Tag) string
}

type IModelI18nEntry interface {
	IModelI18nBase
	GetGlobalId() string
	GetObjType() string
	GetObjId() string
	GetKeyName() string
}

type IModelI18nTable interface {
	GetI18nEntries() map[string]IModelI18nEntry
}

type SModelI18nEntry struct {
	model        IModel
	i18nEntry    IModelI18nBase
	modelKeyName string
}

func NewSModelI18nEntry(keyName string, model IModel, i18nEntry IModelI18nBase) *SModelI18nEntry {
	return &SModelI18nEntry{model: model, modelKeyName: keyName, i18nEntry: i18nEntry}
}

func (self *SModelI18nEntry) GetObjType() string {
	return self.model.Keyword()
}

func (self *SModelI18nEntry) GetObjId() string {
	return self.model.GetId()
}

func (self *SModelI18nEntry) GetKeyName() string {
	return self.modelKeyName
}

func (self *SModelI18nEntry) GetKeyValue() string {
	return self.i18nEntry.GetKeyValue()
}

func (self *SModelI18nEntry) Lookup(tag i18n.Tag) string {
	return self.i18nEntry.Lookup(tag)
}

func (self *SModelI18nEntry) GetGlobalId() string {
	return self.model.Keyword() + "::" + self.model.GetId() + "::" + self.modelKeyName
}

type SModelI18nTable struct {
	i18nEntires []IModelI18nEntry
}

func NewSModelI18nTable(i18nEntires []IModelI18nEntry) *SModelI18nTable {
	return &SModelI18nTable{i18nEntires: i18nEntires}
}

func (t *SModelI18nTable) GetI18nEntries() map[string]IModelI18nEntry {
	ret := make(map[string]IModelI18nEntry, 0)
	for i := range t.i18nEntires {
		entry := t.i18nEntires[i]
		ret[entry.GetKeyValue()] = entry
	}

	return ret
}

type SI18nManager struct {
	SStandaloneAnonResourceBaseManager
}

type SI18n struct {
	SStandaloneAnonResourceBase

	// 资源类型
	// example: network
	ObjType string `width:"40" charset:"ascii" index:"true" list:"user" get:"user"`

	// 资源ID
	// example: 87321a70-1ecb-422a-8b0c-c9aa632a46a7
	ObjId string `width:"88" charset:"ascii" index:"true" list:"user" get:"user"`

	// 资源KEY
	// exmaple: name
	KeyName string `width:"64" charset:"utf8" primary:"true" list:"user" get:"user"`

	// 资源原始值
	// example: 技术部
	KeyValue string `charset:"utf8" list:"user" get:"user"`

	// KeyValue 对应中文翻译
	Cn string `charset:"utf8" list:"user" get:"user" update:"admin_required" update:"admin"`

	// KeyValue 对应英文翻译
	En string `charset:"utf8" list:"user" get:"user" update:"admin_required" update:"admin"`
}

var I18nManager *SI18nManager

func init() {
	I18nManager = &SI18nManager{
		SStandaloneAnonResourceBaseManager: NewStandaloneAnonResourceBaseManager(
			SI18n{},
			"i18n_tbl",
			"i18n",
			"i18ns",
		),
	}
	I18nManager.SetVirtualObject(I18nManager)
}

func (manager *SModelBaseManager) GetModelI18N(ctx context.Context, model IModel) ([]SI18n, error) {
	ret := []SI18n{}
	q := manager.Query().Equals("obj_type", model.Keyword()).Equals("obj_id", model.GetId())
	err := FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}

	return ret, err
}

func (manager *SModelBaseManager) GetModelKeyI18N(ctx context.Context, model IModel, keyName string) ([]SI18n, error) {
	ret := []SI18n{}
	q := manager.Query().Equals("obj_type", model.Keyword()).Equals("obj_id", model.GetId()).Equals("key_name", keyName)
	err := FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}

	return ret, err
}

func (manager *SI18nManager) getExternalI18nItems(ctx context.Context, table IModelI18nTable) []IModelI18nEntry {
	ret := make([]IModelI18nEntry, 0)
	items := table.GetI18nEntries()
	for k, _ := range items {
		ret = append(ret, items[k])
	}

	return ret
}

func (manager *SI18nManager) RemoveI18ns(ctx context.Context, userCred mcclient.TokenCredential, model IModel) error {
	dbItems := []SI18n{}
	q := manager.Query().Equals("obj_type", model.Keyword()).Equals("obj_id", model.GetId())
	err := FetchModelObjects(manager, q, &dbItems)
	if err != nil {
		return err
	}

	for i := 0; i < len(dbItems); i += 1 {
		err = dbItems[i].removeI18n(ctx, userCred)
		if err != nil {
			return err
		}
	}

	return nil
}

func (manager *SI18nManager) SyncI18ns(ctx context.Context, userCred mcclient.TokenCredential, model IModel, table IModelI18nTable) ([]SI18n, []IModelI18nEntry, compare.SyncResult) {
	//// No need to lock SI18nManager, the resources has been lock in the upper layer - QIU Jian, 20210405
	// lockman.LockClass(ctx, manager, "")
	// defer lockman.ReleaseClass(ctx, manager, "")

	syncResult := compare.SyncResult{}

	extItems := manager.getExternalI18nItems(ctx, table)
	dbItems := make([]SI18n, 0)
	q := manager.Query().Equals("obj_type", model.Keyword()).Equals("obj_id", model.GetId())
	err := FetchModelObjects(manager, q, &dbItems)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SI18n, 0)
	commondb := make([]SI18n, 0)
	commonext := make([]IModelI18nEntry, 0)
	added := make([]IModelI18nEntry, 0)

	err = compare.CompareSets(dbItems, extItems, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].removeI18n(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	locals := make([]SI18n, 0)
	remotes := make([]IModelI18nEntry, 0)
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].updateFromI18n(ctx, userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			locals = append(locals, commondb[i])
			remotes = append(remotes, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromI18n(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			locals = append(locals, *new)
			remotes = append(remotes, added[i])
			syncResult.Add()
		}
	}

	return locals, remotes, syncResult
}

func (manager *SI18nManager) newFromI18n(ctx context.Context, userCred mcclient.TokenCredential, entry IModelI18nEntry) (*SI18n, error) {
	in := SI18n{}
	in.SetModelManager(manager, &in)

	in.ObjId = entry.GetObjId()
	in.ObjType = entry.GetObjType()
	in.KeyName = entry.GetKeyName()
	in.KeyValue = entry.GetKeyValue()
	in.Cn = entry.Lookup(i18n.I18N_TAG_CHINESE)
	in.En = entry.Lookup(i18n.I18N_TAG_ENGLISH)

	err := manager.TableSpec().Insert(ctx, &in)
	if err != nil {
		log.Infof("newFromI18n fail %s", err)
		return nil, err
	}

	return &in, nil
}

func (self *SI18n) updateFromI18n(ctx context.Context, userCred mcclient.TokenCredential, entry IModelI18nEntry) error {
	_, err := Update(self, func() error {
		self.KeyValue = entry.GetKeyValue()
		self.Cn = entry.Lookup(i18n.I18N_TAG_CHINESE)
		self.En = entry.Lookup(i18n.I18N_TAG_ENGLISH)

		return nil
	})
	if err != nil {
		log.Infof("updateFromI18n error %s", err)
		return err
	}

	return nil
}

func (self *SI18n) removeI18n(ctx context.Context, userCred mcclient.TokenCredential) error {
	// lockman.LockObject(ctx, self)
	// defer lockman.ReleaseObject(ctx, self)

	err := self.Delete(ctx, userCred)
	return err
}

func (self SI18n) GetExternalId() string {
	return self.ObjType + "::" + self.ObjId + "::" + self.KeyName
}

func (self *SI18n) Lookup(ctx context.Context) string {
	entry := i18n.NewTableEntry().CN(self.Cn).EN(self.En)
	if v, ok := entry.Lookup(ctx); ok {
		return v
	}

	return self.KeyValue
}
