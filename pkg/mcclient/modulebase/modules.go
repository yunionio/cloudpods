// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modulebase

import (
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type IBaseManager interface {
	Version() string
	// GetApiVersion() string
	GetKeyword() string
	KeyString() string
	ServiceType() string
	EndpointType() string
	GetColumns(session *mcclient.ClientSession) []string
	List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*printutils.ListResult, error)
}

type ManagerContext struct {
	InstanceManager Manager
	InstanceId      string
}

type Manager interface {
	IBaseManager
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
	ListInContext(session *mcclient.ClientSession, params jsonutils.JSONObject, ctx Manager, ctxid string) (*printutils.ListResult, error)
	ListInContexts(session *mcclient.ClientSession, params jsonutils.JSONObject, ctxs []ManagerContext) (*printutils.ListResult, error)
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
	BatchGet(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []printutils.SubmitResult
	BatchGetInContext(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctx Manager, ctxid string) []printutils.SubmitResult
	BatchGetInContexts(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctxs []ManagerContext) []printutils.SubmitResult
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
	BatchCreate(session *mcclient.ClientSession, params jsonutils.JSONObject, count int) []printutils.SubmitResult
	BatchCreateInContext(session *mcclient.ClientSession, params jsonutils.JSONObject, count int, ctx Manager, ctxid string) []printutils.SubmitResult
	BatchCreateInContexts(session *mcclient.ClientSession, params jsonutils.JSONObject, count int, ctxs []ManagerContext) []printutils.SubmitResult
	Update(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Put(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PutSpecific(session *mcclient.ClientSession, id string, spec string, query jsonutils.JSONObject, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PutInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	PutInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchUpdate(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []printutils.SubmitResult
	BatchParamsUpdate(session *mcclient.ClientSession, idlist []string, params []jsonutils.JSONObject) []printutils.SubmitResult
	BatchPut(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []printutils.SubmitResult
	BatchPutInContext(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctx Manager, ctxid string) []printutils.SubmitResult
	BatchPutInContexts(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctxs []ManagerContext) []printutils.SubmitResult
	Patch(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PatchInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	PatchInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchPatch(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []printutils.SubmitResult
	BatchPatchInContext(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctx Manager, ctxid string) []printutils.SubmitResult
	BatchPatchInContexts(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctxs []ManagerContext) []printutils.SubmitResult
	PerformAction(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PerformClassAction(session *mcclient.ClientSession, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PerformActionInContext(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	PerformActionInContexts(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchPerformAction(session *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject) []printutils.SubmitResult
	BatchPerformActionInContext(session *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctx Manager, ctxid string) []printutils.SubmitResult
	BatchPerformActionInContexts(session *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctxs []ManagerContext) []printutils.SubmitResult
	Delete(session *mcclient.ClientSession, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
	DeleteWithParam(session *mcclient.ClientSession, id string, query jsonutils.JSONObject, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
	DeleteInContext(session *mcclient.ClientSession, id string, body jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	DeleteInContextWithParam(session *mcclient.ClientSession, id string, query jsonutils.JSONObject, body jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error)
	DeleteInContexts(session *mcclient.ClientSession, id string, body jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	DeleteInContextsWithParam(session *mcclient.ClientSession, id string, query jsonutils.JSONObject, body jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error)
	BatchDelete(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject) []printutils.SubmitResult
	BatchDeleteWithParam(session *mcclient.ClientSession, idlist []string, query jsonutils.JSONObject, body jsonutils.JSONObject) []printutils.SubmitResult
	BatchDeleteInContext(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject, ctx Manager, ctxid string) []printutils.SubmitResult
	BatchDeleteInContextWithParam(session *mcclient.ClientSession, idlist []string, query jsonutils.JSONObject, body jsonutils.JSONObject, ctx Manager, ctxid string) []printutils.SubmitResult
	BatchDeleteInContexts(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject, ctxs []ManagerContext) []printutils.SubmitResult
	BatchDeleteInContextsWithParam(session *mcclient.ClientSession, idlist []string, query jsonutils.JSONObject, body jsonutils.JSONObject, ctxs []ManagerContext) []printutils.SubmitResult

	GetSpecificMethods() sets.String
}

type IResourceManager interface {
	Manager
	GetMetadata(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	SetMetadata(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
}

type JointManager interface {
	IBaseManager
	MasterManager() Manager
	SlaveManager() Manager
	Get(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	ListDescendent(s *mcclient.ClientSession, mid string, params jsonutils.JSONObject) (*printutils.ListResult, error)
	ListDescendent2(s *mcclient.ClientSession, sid string, params jsonutils.JSONObject) (*printutils.ListResult, error)
	ListAscendent(s *mcclient.ClientSession, mid string, params jsonutils.JSONObject) (*printutils.ListResult, error)
	Attach(s *mcclient.ClientSession, mid, sid string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	BatchAttach(s *mcclient.ClientSession, mid string, sids []string, params jsonutils.JSONObject) []printutils.SubmitResult
	BatchAttach2(s *mcclient.ClientSession, mid string, sids []string, params jsonutils.JSONObject) []printutils.SubmitResult
	Detach(s *mcclient.ClientSession, mid, sid string, query jsonutils.JSONObject) (jsonutils.JSONObject, error)
	BatchDetach(s *mcclient.ClientSession, mid string, sids []string) []printutils.SubmitResult
	BatchDetach2(s *mcclient.ClientSession, mid string, sids []string) []printutils.SubmitResult
	Update(s *mcclient.ClientSession, mid, sid string, query jsonutils.JSONObject, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	BatchUpdate(s *mcclient.ClientSession, mid string, sids []string, query jsonutils.JSONObject, params jsonutils.JSONObject) []printutils.SubmitResult
	Patch(s *mcclient.ClientSession, mid, sid string, query jsonutils.JSONObject, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	BatchPatch(s *mcclient.ClientSession, mid string, sids []string, query jsonutils.JSONObject, params jsonutils.JSONObject) []printutils.SubmitResult
}

var (
	modules      map[string][]IBaseManager
	jointModules map[string][]JointManager
)

func _getJointKey(mod1 Manager, mod2 Manager) string {
	return fmt.Sprintf("%s-%s", mod1.KeyString(), mod2.KeyString())
}

func ensureModuleNotRegistered(mod, newMod IBaseManager) {
	modSvcType := mod.ServiceType()
	newModSvcType := newMod.ServiceType()
	if mod == newMod {
		log.Fatalf("Module %#v duplicate registered, service type: %q", mod, modSvcType)
	}
	if modSvcType != newModSvcType {
		log.Fatalf("Module %#v already registered, service type is %q.\nSo new module %#v can't be registered, service type is %q", mod, modSvcType, newMod, newModSvcType)
	}
}

func Register(mod IBaseManager) {
	if modules == nil {
		modules = make(map[string][]IBaseManager)
	}
	mods, ok := modules[mod.KeyString()]
	if !ok {
		mods = make([]IBaseManager, 0)
	}
	for i := range mods {
		ensureModuleNotRegistered(mods[i], mod)
	}
	mods = append(mods, mod)
	modules[mod.KeyString()] = mods
	// modtable[mod.KeyString()] = append(mods, mod)
}

func RegisterJointModule(mod IBaseManager) {
	jointMod, ok := mod.(JointManager)
	if ok { // also a joint manager
		jointKey := _getJointKey(jointMod.MasterManager(), jointMod.SlaveManager())
		// log.Printf("%s(%s) is also a joint module", mod.KeyString(), jointKey)
		jointMods, ok := jointModules[jointKey]
		if !ok {
			jointMods = make([]JointManager, 0)
		}
		for i := range jointMods {
			// if m == jointMod {
			ensureModuleNotRegistered(jointMods[i], jointMod)
			//}
		}
		// modtable[jointKey] = append(jointMods, jointMod)
		jointModules[jointKey] = append(jointMods, jointMod)
	}
}

func registerAllJointModules() {
	if jointModules == nil {
		jointModules = make(map[string][]JointManager)
		for modname := range modules {
			for i := range modules[modname] {
				RegisterJointModule(modules[modname][i])
			}
		}
	}
}

func _getModule(session *mcclient.ClientSession, name string) (IBaseManager, error) {
	mods, ok := modules[name]
	if !ok {
		return nil, fmt.Errorf("No such module %s", name)
	}

	if len(mods) == 1 {
		return mods[0], nil
	}

	for _, mod := range mods {
		url, e := session.GetServiceURL(mod.ServiceType(), mod.EndpointType())
		if e != nil {
			return nil, errors.Wrap(e, "session.GetServiceURL")
		}
		_, ver := mcclient.SplitVersionedURL(url)
		log.Debugf("url: %s ver: %s mod.Version: %s", url, ver, mod.Version())
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
	mods, ok := jointModules[key]
	if !ok {
		return nil, fmt.Errorf("No such joint module: %s", key)
	}
	for _, mod := range mods {
		url, e := session.GetServiceVersionURL(mod.ServiceType(), mod.EndpointType())
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

func GetRegisterdModules() ([]string, []string) {
	registerAllJointModules()

	ret := make([]string, 0)
	for k := range modules {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	ret2 := make([]string, 0)
	for k := range jointModules {
		ret2 = append(ret2, k)
	}
	sort.Strings(ret2)
	return ret, ret2
}
