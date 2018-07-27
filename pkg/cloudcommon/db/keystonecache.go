package db

type SKeystoneCacheObjectManager struct {
	SStandaloneResourceBaseManager
}

type SKeystoneCacheObject struct {
	SStandaloneResourceBase

	DomainId string `width:"128" charset:"ascii" nullable:"true"`
	Domain   string `width:"128" charset:"utf8" nullable:"true"`
}

func NewKeystoneCacheObjectManager(dt interface{}, tableName string, keyword string, keywordPlural string) SKeystoneCacheObjectManager {
	return SKeystoneCacheObjectManager{SStandaloneResourceBaseManager: NewStandaloneResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func NewKeystoneCacheObject(id string, name string, domainId string, domain string) SKeystoneCacheObject {
	obj := SKeystoneCacheObject{}
	obj.Id = id
	obj.Name = name
	obj.Domain = domain
	obj.DomainId = domainId
	return obj
}

/*func (manager *SKeystoneCacheObjectManager) Save(ctx context.Context, idStr string, name string, domainId string, domain string) (*SKeystoneCacheObject, error) {
	lockman.LockRawObject(ctx, manager.keyword, idStr)
	defer lockman.ReleaseRawObject(ctx, manager.keyword, idStr)

	modelObjm, err := manager.FetchById(idStr)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		obj := NewKeystoneCacheObject(idStr, name, domainId, domain)

		err = manager.TableSpec().Insert(&obj)
		if err != nil {
			return nil, err
		}
	}

	modelObjm, err = manager.FetchById(idStr)
	if err != nil {
		return nil, err
	}

	objm := modelObjm.(*SKeystoneCacheObject)
	_, err = manager.TableSpec().Update(objm, func() error {
		reflectutils.FillEmbededStructValue()
		objm.Id = idStr
		objm.Name = name
		objm.DomainId = domainId
		objm.Domain = domain
		return nil
	})
	if err != nil {
		return nil, err
	} else {
		return objm, nil
	}
}*/

func (manager *SKeystoneCacheObjectManager) BatchFetchNames(idStrs []string) []string {
	t := manager.tableSpec.Instance()
	results, err := t.Query(t.Field("name")).In("id", idStrs).AllStringMap()
	if err != nil {
		return nil
	}
	ret := make([]string, len(results))
	for i, obj := range results {
		ret[i] = obj["name"]
	}
	return ret
}

/* @classmethod
def is_idstr(cls, idstr):
return not stringutils.is_chs(idstr)

@classmethod
def fetch_by_id_or_name(cls, idstr):
obj = cls.fetch_by_id(idstr)
if obj is None:
obj = cls.fetch_by_name(idstr, None)
return obj

@classmethod
def fetch_by_name(cls, namestr, user_cred):
ret = cls.query().filter(cls.name==namestr).all()
if len(ret) == 0:
return None
elif len(ret) == 1:
return ret[0]
else:
raise Exception("Duplicate name %s" % namestr) */
