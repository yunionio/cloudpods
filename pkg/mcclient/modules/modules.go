package modules

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type BaseManagerInterface interface {
	Version() string
	GetApiVersion() string
	GetKeyword() string
	KeyString() string
	ServiceType() string
	EndpointType() string
	GetColumns(session *mcclient.ClientSession) []string
	List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*ListResult, error)
	SetApiVersion(string)
}

type ManagerContext struct {
	InstanceManager Manager
	InstanceId      string
}

type Manager interface {
	BaseManagerInterface
	/* resource list
	   GET <base_url>/<resource_plural_keyword>
	   e.g GET <base_url>/alarms
	   querystring stores in params
	   e.g. GET <base_url>/alarms?limit=20&offset=20&search=test

	   return list of resources in json format
	   { "<resource_plural_keyword>": [ {object details}, {object details}, ...] }, limit: 20, offset: 20, total: 2000}
	*/
	// List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*ListResult, error)
	/*
	   resource list in a context
	   GET <base_url>/<context_plural_keyword>/<context_id>/<resource_plural_keyword>?querystring
	   e.g. GET <base_url>/nodes/1/labels?name=xxx
	   ListInContext(s, params, &modules.Labels, label_id)
	   return:
	   { "<resource_plural_keyword>": [ {object details}, {object details}, ...] }, limit: 20, offset: 20, total: 2000}
	*/
	ListInContext(session *mcclient.ClientSession, params jsonutils.JSONObject, ctx Manager, ctxid string) (*ListResult, error)
	ListInContexts(session *mcclient.ClientSession, params jsonutils.JSONObject, ctxs []ManagerContext) (*ListResult, error)
	/*
	  GET <base_url>/<resource_plural_keyword>/<resource_id>
	  e.g GET <base_url>/alarams/1

	*/
	Get(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	GetInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	GetInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	GetId(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (string, error)
	GetIdInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (string, error)
	GetIdInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (string, error)
	GetById(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	GetByIdInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	GetByIdInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	GetByName(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	GetByNameInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	GetByNameInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	/*
	  HEAD <base_url>/<resource_plural_keyword>/<resource_id>
	  e.g HEAD <base_url>/alarams/1

	*/
	Head(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	HeadInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	HeadInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchGet(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []SubmitResult
	BatchGetInContext(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult
	BatchGetInContexts(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult
	GetSpecific(session *mcclient.ClientSession, id string, spec string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	GetSpecificInContext(session *mcclient.ClientSession, id string, spec string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	GetSpecificInContexts(session *mcclient.ClientSession, id string, spec string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	/*
	  POST <base_url>/<resource_plural_keyword>/<resource_id>
	  e.g POST <base_url>/alarams/1

	*/
	Create(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	CreateInContext(session *mcclient.ClientSession, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	CreateInContexts(session *mcclient.ClientSession, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchCreate(session *mcclient.ClientSession, params jsonutils.JSONObject, count int) []SubmitResult
	BatchCreateInContext(session *mcclient.ClientSession, params jsonutils.JSONObject, count int, ctx Manager, ctxid string) []SubmitResult
	BatchCreateInContexts(session *mcclient.ClientSession, params jsonutils.JSONObject, count int, ctxs []ManagerContext) []SubmitResult
	Update(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Put(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PutInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	PutInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchUpdate(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []SubmitResult
	BatchPut(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []SubmitResult
	BatchPutInContext(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult
	BatchPutInContexts(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult
	Patch(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PatchInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	PatchInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchPatch(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []SubmitResult
	BatchPatchInContext(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult
	BatchPatchInContexts(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult
	PerformAction(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PerformClassAction(session *mcclient.ClientSession, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PerformActionInContext(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	PerformActionInContexts(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchPerformAction(session *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject) []SubmitResult
	BatchPerformActionInContext(session *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult
	BatchPerformActionInContexts(session *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult
	Delete(session *mcclient.ClientSession, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
	DeleteWithParam(session *mcclient.ClientSession, id string, query jsonutils.JSONObject, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
	DeleteInContext(session *mcclient.ClientSession, id string, body jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	DeleteInContextWithParam(session *mcclient.ClientSession, id string, query jsonutils.JSONObject, body jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	DeleteInContexts(session *mcclient.ClientSession, id string, body jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	DeleteInContextsWithParam(session *mcclient.ClientSession, id string, query jsonutils.JSONObject, body jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchDelete(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject) []SubmitResult
	BatchDeleteWithParam(session *mcclient.ClientSession, idlist []string, query jsonutils.JSONObject, body jsonutils.JSONObject) []SubmitResult
	BatchDeleteInContext(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult
	BatchDeleteInContextWithParam(session *mcclient.ClientSession, idlist []string, query jsonutils.JSONObject, body jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult
	BatchDeleteInContexts(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult
	BatchDeleteInContextsWithParam(session *mcclient.ClientSession, idlist []string, query jsonutils.JSONObject, body jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult
}

type JointManager interface {
	BaseManagerInterface
	MasterManager() Manager
	SlaveManager() Manager
	Get(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	ListDescendent(s *mcclient.ClientSession, mid string, params jsonutils.JSONObject) (*ListResult, error)
	ListDescendent2(s *mcclient.ClientSession, sid string, params jsonutils.JSONObject) (*ListResult, error)
	ListAscendent(s *mcclient.ClientSession, mid string, params jsonutils.JSONObject) (*ListResult, error)
	Attach(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	BatchAttach(s *mcclient.ClientSession, mid string, sids []string, params jsonutils.JSONObject) []SubmitResult
	BatchAttach2(s *mcclient.ClientSession, mid string, sids []string, params jsonutils.JSONObject) []SubmitResult
	Detach(s *mcclient.ClientSession, mid, sid string, query jsonutils.JSONObject) (jsonutils.JSONObject, error)
	BatchDetach(s *mcclient.ClientSession, mid string, sids []string) []SubmitResult
	BatchDetach2(s *mcclient.ClientSession, mid string, sids []string) []SubmitResult
	Update(s *mcclient.ClientSession, mid, sid string, query jsonutils.JSONObject, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	BatchUpdate(s *mcclient.ClientSession, mid string, sids []string, query jsonutils.JSONObject, params jsonutils.JSONObject) []SubmitResult
	Patch(s *mcclient.ClientSession, mid, sid string, query jsonutils.JSONObject, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	BatchPatch(s *mcclient.ClientSession, mid string, sids []string, query jsonutils.JSONObject, params jsonutils.JSONObject) []SubmitResult
}

var (
	modules      map[string]map[string][]BaseManagerInterface
	jointModules map[string]map[string][]JointManager
)

func _getJointKey(mod1 Manager, mod2 Manager) string {
	return fmt.Sprintf("%s-%s", mod1.KeyString(), mod2.KeyString())
}

func ensureModuleNotRegistered(mod, newMod BaseManagerInterface) {
	modSvcType := mod.ServiceType()
	newModSvcType := newMod.ServiceType()
	if mod == newMod {
		log.Fatalf("Module %#v duplicate registered, service type: %q", mod, modSvcType)
	}
	if modSvcType != newModSvcType {
		log.Fatalf("Module %#v already registered, service type is %q.\nSo new module %#v can't be registered, service type is %q", mod, modSvcType, newMod, newModSvcType)
	}
}

func _register(version string, mod BaseManagerInterface) {
	if modules == nil {
		modules = make(map[string]map[string][]BaseManagerInterface)
	}
	modtable, ok := modules[version]
	if !ok {
		modtable = make(map[string][]BaseManagerInterface)
		modules[version] = modtable
	}
	mods, ok := modtable[mod.KeyString()]
	if !ok {
		mods = make([]BaseManagerInterface, 0)
	}
	for _, m := range mods {
		ensureModuleNotRegistered(m, mod)
	}
	modtable[mod.KeyString()] = append(mods, mod)
}

func _registerJointModule(version string, mod BaseManagerInterface) {
	jointMod, ok := mod.(JointManager)
	if ok { // also a joint manager
		jointKey := _getJointKey(jointMod.MasterManager(), jointMod.SlaveManager())
		// log.Printf("%s(%s) is also a joint module", mod.KeyString(), jointKey)
		modtable, ok := jointModules[version]
		if !ok {
			modtable = make(map[string][]JointManager)
			jointModules[version] = modtable
		}
		jointMods, ok := modtable[jointKey]
		if !ok {
			jointMods = make([]JointManager, 0)
		}
		for _, m := range jointMods {
			if m == jointMod {
				ensureModuleNotRegistered(m, jointMod)
			}
		}
		modtable[jointKey] = append(jointMods, jointMod)
	}
}

func registerAllJointModules() {
	if jointModules == nil {
		jointModules = make(map[string]map[string][]JointManager)
		for version, modtable := range modules {
			for _, mods := range modtable {
				for _, mod := range mods {
					_registerJointModule(version, mod)
				}
			}
		}
	}
}

func _getModule(session *mcclient.ClientSession, name string) (BaseManagerInterface, error) {
	modtable, ok := modules[session.GetApiVersion()]
	if !ok {
		return nil, fmt.Errorf("No such version: %s", session.GetApiVersion())
	}
	mods, ok := modtable[name]
	if !ok {
		return nil, fmt.Errorf("No such module %s for version %s", name, session.GetApiVersion())
	}
	for _, mod := range mods {
		url, e := session.GetServiceURL(mod.ServiceType(), mod.EndpointType())
		if e != nil {
			return nil, e
		}
		_, ver := mcclient.SplitVersionedURL(url)
		if strings.EqualFold(ver, mod.Version()) {
			return mod, nil
		}
	}
	return nil, fmt.Errorf("Version mismatch")
}

func GetModule(session *mcclient.ClientSession, name string) (Manager, error) {
	bm, e := _getModule(session, name)
	if e != nil {
		return nil, e
	}
	m, ok := bm.(Manager)
	if ok {
		return m, nil
	} else {
		return nil, fmt.Errorf("Module %s not a Manager", name)
	}
}

func GetJointModule(session *mcclient.ClientSession, name string) (JointManager, error) {
	bm, e := _getModule(session, name)
	if e != nil {
		return nil, e
	}
	m, ok := bm.(JointManager)
	if ok {
		return m, nil
	} else {
		return nil, fmt.Errorf("Module %s not a Manager", name)
	}
}

func GetJointModule2(session *mcclient.ClientSession, mod1 Manager, mod2 Manager) (JointManager, error) {
	registerAllJointModules()
	key := _getJointKey(mod1, mod2)
	modtable, ok := jointModules[session.GetApiVersion()]
	if !ok {
		return nil, fmt.Errorf("No such version: %s", session.GetApiVersion())
	}
	mods, ok := modtable[key]
	if !ok {
		return nil, fmt.Errorf("No such joint module: %s", key)
	}
	for _, mod := range mods {
		url, e := session.GetServiceVersionURL(mod.ServiceType(), mod.EndpointType(), mod.GetApiVersion())
		if e != nil {
			return nil, e
		}
		_, ver := mcclient.SplitVersionedURL(url)
		if strings.EqualFold(ver, mod.Version()) {
			return mod, nil
		}
	}
	return nil, fmt.Errorf("Version mismatch")
}
