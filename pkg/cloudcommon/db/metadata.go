package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

const (
	SYSTEM_ADMIN_PREFIX = "__sys_"
)

type SMetadataManager struct {
	SModelBaseManager
}

type SMetadata struct {
	SModelBase

	Id        string    `width:"128" charset:"ascii" primary:"true"` // = Column(VARCHAR(128, charset='ascii'), primary_key=True)
	Key       string    `width:"64" charset:"ascii" primary:"true"`  // = Column(VARCHAR(64, charset='ascii'),  primary_key=True)
	Value     string    `charset:"utf8"`                             // = Column(TEXT(charset='utf8'), nullable=True)
	UpdatedAt time.Time `nullable:"false" updated_at:"true"`         // = Column(DateTime, default=get_utcnow, nullable=False, onupdate=get_utcnow)
}

var Metadata *SMetadataManager

func init() {
	Metadata = &SMetadataManager{SModelBaseManager: NewModelBaseManager(SMetadata{}, "metadata_tbl", "metadata", "metadata")}
}

func (m *SMetadata) GetId() string {
	return fmt.Sprintf("%s-%s", m.Id, m.Key)
}

func (m *SMetadata) GetName() string {
	return fmt.Sprintf("%s-%s", m.Id, m.Key)
}

func (m *SMetadata) GetModelManager() IModelManager {
	return Metadata
}

func GetObjectIdstr(model IModel) string {
	return fmt.Sprintf("%s::%s", model.GetModelManager().Keyword(), model.GetId())
}

/* @classmethod
def get_object_idstr(cls, obj, keygen_func):
idstr = None
if keygen_func is not None and callable(keygen_func):
idstr = keygen_func(obj)
elif isinstance(obj, SStandaloneResourceBase):
idstr = '%s::%s' % (obj._resource_name_, obj.id)
if idstr is None:
raise Exception('get_object_idstr: failed to generate obj ID')
return idstr */

func (manager *SMetadataManager) GetStringValue(model IModel, key string, userCred mcclient.TokenCredential) string {
	if strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX) && (userCred == nil || !IsAdminAllowGetSpec(userCred, model, "metadata")) {
		return ""
	}
	idStr := GetObjectIdstr(model)
	m := SMetadata{}
	err := manager.Query().Equals("id", idStr).Equals("key", key).First(&m)
	if err == nil {
		return m.Value
	}
	return ""
}

func (manager *SMetadataManager) GetJsonValue(model IModel, key string, userCred mcclient.TokenCredential) jsonutils.JSONObject {
	if strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX) && (userCred == nil || !IsAdminAllowGetSpec(userCred, model, "metadata")) {
		return nil
	}
	idStr := GetObjectIdstr(model)
	m := SMetadata{}
	err := manager.Query().Equals("id", idStr).Equals("key", key).First(&m)
	if err == nil {
		json, _ := jsonutils.ParseString(m.Value)
		return json
	}
	return nil
}

type sMetadataChange struct {
	Key    string
	OValue string
	NValue string
}

func (manager *SMetadataManager) RemoveAll(ctx context.Context, model IModel, userCred mcclient.TokenCredential) error {
	idStr := GetObjectIdstr(model)
	if len(idStr) == 0 {
		return fmt.Errorf("invalid model")
	}

	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)

	records := make([]SMetadata, 0)
	q := manager.Query().Equals("id", idStr)
	err := FetchModelObjects(manager, q, &records)
	if err != nil {
		return fmt.Errorf("find metadata for %s fail: %s", idStr, err)
	}
	changes := make([]sMetadataChange, 0)
	for _, rec := range records {
		if len(rec.Value) > 0 {
			_, err := Update(&rec, func() error {
				rec.Value = ""
				return nil
			})
			if err == nil {
				changes = append(changes, sMetadataChange{Key: rec.Key, OValue: rec.Value})
			}
		}
	}
	if len(changes) > 0 {
		OpsLog.LogEvent(model, ACT_DEL_METADATA, jsonutils.Marshal(changes), userCred)
	}
	return nil
}

func (manager *SMetadataManager) SetValue(ctx context.Context, obj IModel, key string, value interface{}, userCred mcclient.TokenCredential) error {
	return manager.SetAll(ctx, obj, map[string]interface{}{key: value}, userCred)
}

func (manager *SMetadataManager) SetAll(ctx context.Context, obj IModel, store map[string]interface{}, userCred mcclient.TokenCredential) error {
	idStr := GetObjectIdstr(obj)

	lockman.LockObject(ctx, obj)
	defer lockman.ReleaseObject(ctx, obj)

	changes := make([]sMetadataChange, 0)
	for key, value := range store {
		valStr := stringutils.Interface2String(value)
		valStrLower := strings.ToLower(valStr)
		if valStrLower == "none" || valStrLower == "null" {
			valStr = ""
		}
		record := SMetadata{}
		err := manager.Query().Equals("id", idStr).Equals("key", key).First(&record)
		if err != nil {
			if err == sql.ErrNoRows {
				changes = append(changes, sMetadataChange{Key: key, NValue: valStr})
				record.Id = idStr
				record.Key = key
				record.Value = valStr
				err = manager.TableSpec().Insert(&record)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			_, err := Update(&record, func() error {
				record.Value = valStr
				return nil
			})
			if err != nil {
				return err
			}
			changes = append(changes, sMetadataChange{Key: key, OValue: record.Value, NValue: valStr})
		}
	}
	if len(changes) > 0 {
		OpsLog.LogEvent(obj, ACT_SET_METADATA, jsonutils.Marshal(changes), userCred)
	}
	return nil
}

func (manager *SMetadataManager) GetAll(obj IModel, keys []string, userCred mcclient.TokenCredential) (map[string]string, error) {
	idStr := GetObjectIdstr(obj)
	records := make([]SMetadata, 0)
	q := manager.Query().Equals("id", idStr)
	if keys != nil && len(keys) > 0 {
		q = q.In("key", keys)
	}
	err := FetchModelObjects(manager, q, &records)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]string)
	for _, rec := range records {
		if len(rec.Value) > 0 {
			if strings.HasPrefix(rec.Key, SYSTEM_ADMIN_PREFIX) {
				if userCred != nil && IsAdminAllowGetSpec(userCred, obj, "metadata") {
					key := rec.Key[len(SYSTEM_ADMIN_PREFIX):]
					ret[key] = rec.Value
				}
			} else {
				ret[rec.Key] = rec.Value
			}
		}
	}
	return ret, nil
}

func (manager *SMetadataManager) IsSystemAdminKey(key string) bool {
	return strings.HasPrefix(key, SYSTEM_ADMIN_PREFIX)
}

func (manager *SMetadataManager) GetSysadminKey(key string) string {
	return fmt.Sprintf("%s%s", SYSTEM_ADMIN_PREFIX, key)
}

/*

@classmethod
def get_sysadmin_key_object_ids(cls, obj_cls, key):
sys_key = cls.get_sysadmin_key(key)
ids = Metadata.query(Metadata.id).filter(Metadata.key==sys_key) \
.filter(Metadata.value!=None) \
.filter(Metadata.id.like('%s::%%' % obj_cls._resource_name_)) \
.all()
ret = []
for id, in ids:
ret.append(id[len(obj_cls._resource_name_)+2:])
return ret

*/
