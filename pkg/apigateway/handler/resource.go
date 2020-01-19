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

package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/excelutils"
)

const (
	APIVer   = "<apiver>"
	ResName  = "<resname>"
	ResID    = "<resid>"
	ResName2 = "<resname2>"
	ResID2   = "<resid2>"
	ResName3 = "<resname3>"
	ResID3   = "<resid3>"
	Spec     = "<spec>"
	Action   = "<action>"

	GET    = "GET"
	PUT    = "PUT"
	POST   = "POST"
	DELETE = "DELETE"
	PATCH  = "PATCH"
)

type ResourceHandlers struct {
	*SHandlers
}

func NewResourceHandlers(prefix string) *ResourceHandlers {
	return &ResourceHandlers{NewHandlers(prefix)}
}

func (f *ResourceHandlers) AddGet(mf appsrv.MiddlewareFunc) *ResourceHandlers {
	hs := []HandlerPath{
		// list and joint list
		NewHP(f.listHandler, APIVer, ResName),
		// get
		NewHP(f.getHandler, APIVer, ResName, ResID),
		// get spec, list in context, joint list descendent
		NewHP(f.getSpecHandler, APIVer, ResName, ResID, Spec),
		// joint get
		NewHP(f.getJointHandler, APIVer, ResName, ResID, ResName2, ResID2),
		// list in 2 context
		NewHP(f.listInContextsHandler, APIVer, ResName, ResID, ResName2, ResID2, ResName3),
	}
	f.AddByMethod(GET, mf, hs...)
	return f
}

func (f *ResourceHandlers) AddPost(mf appsrv.MiddlewareFunc) *ResourceHandlers {
	hs := []HandlerPath{
		// create, create multi
		NewHP(f.createHandler, APIVer, ResName),
		// batch performAction
		NewHP(f.batchPerformActionHandler, APIVer, ResName, Action),
		// performAction, create in context, batchAttach
		NewHP(f.performActionHandler, APIVer, ResName, ResID, Action),
		// joint attach
		NewHP(f.attachHandler, APIVer, ResName, ResID, ResName2, ResID2),
	}
	f.AddByMethod(POST, mf, hs...)
	return f
}

func (f *ResourceHandlers) AddPut(mf appsrv.MiddlewareFunc) *ResourceHandlers {
	hs := []HandlerPath{
		// batchPut
		NewHP(f.batchUpdateHandler, APIVer, ResName),
		// put
		NewHP(f.updateHandler, APIVer, ResName, ResID),
		// batchPut joint
		NewHP(f.batchUpdateJointHandler, APIVer, ResName, ResID, ResName2),
		// update joint, update in context
		NewHP(f.updateJointHandler, APIVer, ResName, ResID, ResName2, ResID2),
		// update in contexts
		NewHP(f.updateInContextsHandler, APIVer, ResName, ResID, ResName2, ResID2, ResName3, ResID3),
	}
	f.AddByMethod(PUT, mf, hs...)
	return f
}

func (f *ResourceHandlers) AddPatch(mf appsrv.MiddlewareFunc) *ResourceHandlers {
	hs := []HandlerPath{
		// batchPatch
		NewHP(f.batchPatchHandler, APIVer, ResName),
		// patch
		NewHP(f.patchHandler, APIVer, ResName, ResID),
		// patch joint, patch in context
		NewHP(f.patchJointHandler, APIVer, ResName, ResID, ResName2, ResID2),
		// patch in contexts
		NewHP(f.patchInContextsHandler, APIVer, ResName, ResID, ResName2, ResID2, ResName3, ResID3),
	}
	f.AddByMethod(PATCH, mf, hs...)
	return f
}

func (f *ResourceHandlers) AddDelete(mf appsrv.MiddlewareFunc) *ResourceHandlers {
	hs := []HandlerPath{
		// batchDelete
		NewHP(f.batchDeleteHandler, APIVer, ResName),
		// delete
		NewHP(f.deleteHandler, APIVer, ResName, ResID),
		// batch detach
		NewHP(f.batchDetachHandler, APIVer, ResName, ResID, ResName2),
		// detach joint, delete in context
		NewHP(f.detachHandle, APIVer, ResName, ResID, ResName2, ResID2),
		// list in 2 context
		NewHP(f.deleteInContextsHandler, APIVer, ResName, ResID, ResName2, ResID2, ResName3, ResID3),
	}
	f.AddByMethod(DELETE, mf, hs...)
	return f
}

func fetchIdList(query jsonutils.JSONObject, w http.ResponseWriter) []string {
	idlist, e := query.GetArray("id")
	if e == nil && len(idlist) > 0 {
		queryDict := query.(*jsonutils.JSONDict)
		queryDict.Remove("id")
		log.Debugf("Get idlist: %s", idlist)
		return jsonutils.JSONArray2StringArray(idlist)
	} else {
		log.Debugf("Cannot find idlist in query: %s", query)
		httperrors.InvalidInputError(w, "No idlist found")
		return nil
	}
}

// list
// joint list
// /<resname>
func (f *ResourceHandlers) listHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r)
	if req.Error() != nil {
		httperrors.GeneralServerError(w, req.Error())
		return
	}
	jmod, _ := modulebase.GetJointModule(req.Session(), req.ResName())
	if jmod != nil {
		f.doList(req.Session(), jmod, req.Query(), w, r)
	} else {
		if err := req.WithMod1().Error(); err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
		batchGet, err := req.Query().Bool("batchGet")
		if err == nil && batchGet {
			idlist := fetchIdList(req.Query(), w)
			if idlist == nil {
				return
			}
			querydict, ok := req.Query().(*jsonutils.JSONDict)
			if ok {
				querydict.Remove("batchGet")
				querydict.Remove("id")
			}
			ret := req.Mod1().BatchGet(req.Session(), idlist, req.Query())
			appsrv.SendJSON(w, modulebase.ListResult2JSON(modulebase.SubmitResults2ListResult(ret)))
		} else {
			f.doList(req.Session(), req.Mod1(), req.Query(), w, r)
		}
	}
}

func (f *ResourceHandlers) doList(session *mcclient.ClientSession, module modulebase.IBaseManager, query jsonutils.JSONObject, w http.ResponseWriter, r *http.Request) {
	var exportKeys []string
	var exportTexts []string
	exportFormat, _ := query.GetString("export")
	if len(exportFormat) > 0 {
		exportKeyStr, _ := query.GetString("export_keys")
		exportTextStr, _ := query.GetString("export_texts")
		if len(exportKeyStr) > 0 {
			exportKeys = strings.Split(exportKeyStr, ",")
		}
		if len(exportTextStr) > 0 {
			exportTexts = strings.Split(exportTextStr, ",")
		}
		if len(exportKeys) == 0 {
			httperrors.InvalidInputError(w, "missing export keys")
			return
		} else if len(exportKeys) != len(exportTexts) {
			if len(exportTexts) == 0 {
				exportTexts = exportKeys
			} else {
				httperrors.InvalidInputError(w, "inconsistent export keys and texts")
				return
			}
		}
		queryDict := query.(*jsonutils.JSONDict)
		queryDict.Remove("export")
		queryDict.Remove("export_texts")
		query = queryDict
	}

	ret, e := module.List(session, query)
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else if len(exportFormat) > 0 {
		w.Header().Set("Content-Description", "File Transfer")
		w.Header().Set("Content-Transfer-Encoding", "binary")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"export-%s.xlsx\"", module.KeyString()))
		excelutils.Export(ret.Data, exportKeys, exportTexts, w)
	} else {
		appsrv.SendJSON(w, modulebase.ListResult2JSON(ret))
	}
}

// get single
// /<resname>/<resid>
func (f *ResourceHandlers) getHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	obj, e := req.Mod1().Get(req.Session(), req.ResID(), req.Query())
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

// * get spec
// * list in context
// * joint list descendent
// /<resname>/<resid>/<spec>
// /<resname>/<resid>/<resname2>
func (f *ResourceHandlers) getSpecHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	session := req.Session()
	module := req.Mod1()
	query := req.Query()

	module2, e := modulebase.GetModule(session, req.Spec())
	if e == nil { // list in 1 context
		jmod, e := modulebase.GetJointModule2(session, module, module2)
		var ret *modulebase.ListResult
		if e == nil { // joint module
			ret, e = jmod.ListDescendent(session, req.ResID(), query)
		} else {
			ret, e = module2.ListInContext(session, query, module, req.ResID())
		}
		if e != nil {
			httperrors.GeneralServerError(w, e)
		} else {
			appsrv.SendJSON(w, modulebase.ListResult2JSON(ret))
		}
	} else {
		obj, e := module.GetSpecific(session, req.ResID(), req.Spec(), query)
		if e != nil {
			httperrors.GeneralServerError(w, e)
		} else {
			appsrv.SendJSON(w, obj)
		}
	}
}

// joint get
// /<resname>/<resid>/<resname2>/<resid2>
func (f *ResourceHandlers) getJointHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	module := req.Mod1()
	module2 := req.Mod2()
	session := req.Session()

	jmod, e := modulebase.GetJointModule2(session, module, module2)
	if e != nil {
		httperrors.NotFoundError(w, fmt.Sprintf("resource %s-%s not exist", req.ResName(), req.ResName2()))
		return
	}
	obj, e := jmod.Get(session, req.ResID(), req.ResID2(), req.Query())
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

// list in 2 context
// /<resname>/<resid>/<resname2>/<resid2>/<resnam3>
func (f *ResourceHandlers) listInContextsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2().WithMod3()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	module := req.Mod1()
	module2 := req.Mod2()
	module3 := req.Mod3()
	session := req.Session()
	query := req.Query()
	obj, e := module3.ListInContexts(session, query, []modulebase.ManagerContext{{module, req.ResID()}, {module2, req.ResID2()}})
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, modulebase.ListResult2JSON(obj))
	}
}

// create
// create multi
// /<resname>
func (f *ResourceHandlers) createHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	module := req.Mod1()
	session := req.Session()
	body := req.Body()

	count, e := body.Int("__count__")
	if e == nil && count > 1 {
		bodyDict, ok := body.(*jsonutils.JSONDict)
		if !ok {
			httperrors.GeneralServerError(w, fmt.Errorf("Fail to decode body"))
		} else {
			nbody := bodyDict.Copy("__count__")
			ret := module.BatchCreate(session, nbody, int(count))
			w.WriteHeader(207)
			appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
		}
	} else {
		obj, e := module.Create(session, body)
		if e != nil {
			httperrors.GeneralServerError(w, e)
		} else {
			appsrv.SendJSON(w, obj)
		}
	}
}

// batch performAction
// /<resname>/<action>
func (f *ResourceHandlers) batchPerformActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	module := req.Mod1()
	session := req.Session()
	query := req.Query()
	body := req.Body()

	if idlist, e := query.GetArray("id"); e != nil || len(idlist) == 0 {
		if obj, e := module.PerformClassAction(session, req.Action(), body); e != nil {
			httperrors.GeneralServerError(w, e)
		} else {
			appsrv.SendJSON(w, obj)
		}
	} else {
		ret := module.BatchPerformAction(session, jsonutils.JSONArray2StringArray(idlist), req.Action(), body)
		w.WriteHeader(207)
		appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
	}
}

// performAction
// create in context
// batch Attach
// /<resname>/<resid>/<action>
func (f *ResourceHandlers) performActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	module := req.Mod1()
	session := req.Session()
	query := req.Query()
	body := req.Body()

	var obj jsonutils.JSONObject
	var idlist []string
	if module2, e := modulebase.GetModule(session, req.Action()); e == nil {
		if jmod, e := modulebase.GetJointModule2(session, module, module2); e == nil {
			if idlist = fetchIdList(query, w); idlist == nil {
				return
			}
			ret := jmod.BatchAttach(session, req.ResID(), idlist, body)
			w.WriteHeader(207)
			appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
		} else if jmod, e := modulebase.GetJointModule2(session, module2, module); e == nil {
			if idlist = fetchIdList(query, w); idlist == nil {
				return
			}
			ret := jmod.BatchAttach2(session, req.ResID(), idlist, body)
			w.WriteHeader(207)
			appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
		} else {
			if obj, e = module2.CreateInContext(session, body, module, req.ResID()); e != nil {
				httperrors.GeneralServerError(w, e)
			} else {
				appsrv.SendJSON(w, obj)
			}
		}
	} else {
		if obj, e = module.PerformAction(session, req.ResID(), req.Action(), body); e != nil {
			httperrors.GeneralServerError(w, e)
		} else {
			appsrv.SendJSON(w, obj)
		}
	}
}

// joint attach
// /<resname>/<resid>/<resname2>/<resid2>
func (f *ResourceHandlers) attachHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	module := req.Mod1()
	module2 := req.Mod2()
	session := req.Session()
	body := req.Body()

	jmod, e := modulebase.GetJointModule2(session, module, module2)
	if e != nil {
		httperrors.NotFoundError(w, fmt.Sprintf("resource %s-%s not exists", req.ResName(), req.ResName2()))
		return
	}
	obj, e := jmod.Attach(session, req.ResID(), req.ResID2(), body)
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

// batchPut
// PUT /<resname>
func (f *ResourceHandlers) batchUpdateHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	module := req.Mod1()
	query := req.Query()
	body := req.Body()

	idlist := fetchIdList(query, w)
	if idlist == nil {
		return
	}

	ret := module.BatchUpdate(req.Session(), idlist, body)
	w.WriteHeader(207)
	appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
}

// batchPatch
// PATCH /<resname>/<resid>
func (f *ResourceHandlers) batchPatchHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	module := req.Mod1()
	query := req.Query()
	body := req.Body()

	idlist := fetchIdList(query, w)
	if idlist == nil {
		return
	}

	ret := module.BatchPatch(req.Session(), idlist, body)
	w.WriteHeader(207)
	appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
}

// put single
// PUT /<resname>/<resid>
func (f *ResourceHandlers) updateHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	module := req.Mod1()
	body := req.Body()

	obj, e := module.Update(req.Session(), req.ResID(), body)
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

// patch single
// PATCH /<resname>/<resid>
func (f *ResourceHandlers) patchHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	module := req.Mod1()
	body := req.Body()

	obj, e := module.Patch(req.Session(), req.ResID(), body)
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

// update joint
// update in context
// PUT /<resname>/<resid>/<resname2>/<resid2>
func (f *ResourceHandlers) updateJointHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	session := req.Session()
	module := req.Mod1()
	module2 := req.Mod2()
	body := req.Body()

	jmod, e := modulebase.GetJointModule2(session, module, module2)
	var obj jsonutils.JSONObject
	if e == nil { // update joint
		obj, e = jmod.Update(session, req.ResID(), req.ResID2(), nil, body)
	} else { // update in context
		obj, e = module2.PutInContext(session, req.ResID2(), body, module, req.ResID())
	}
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

func (f *ResourceHandlers) patchJointHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	session := req.Session()
	module := req.Mod1()
	module2 := req.Mod2()
	body := req.Body()

	jmod, e := modulebase.GetJointModule2(session, module, module2)
	var obj jsonutils.JSONObject
	if e == nil { // update joint
		obj, e = jmod.Patch(session, req.ResID(), req.ResID2(), nil, body)
	} else { // update in context
		obj, e = module2.PatchInContext(session, req.ResID2(), body, module, req.ResID())
	}
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

// batch update Joint
func (f *ResourceHandlers) batchUpdateJointHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	session := req.Session()
	module := req.Mod1()
	module2 := req.Mod2()
	body := req.Body()
	query := req.Query()

	idlist := fetchIdList(query, w)
	if idlist == nil {
		return
	}
	jmod, e := modulebase.GetJointModule2(session, module, module2)
	if e != nil { // update joint
		httperrors.GeneralServerError(w, e)
		return
	}
	ret := jmod.BatchUpdate(session, req.ResID(), idlist, query, body)
	w.WriteHeader(207)
	appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
}

// update in contexts
// /<resname>/<resid>/<resname2>/<resid2>/<resname3>/<resid3>
func (f *ResourceHandlers) updateInContextsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2().WithMod3()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	obj, e := req.Mod3().PutInContexts(req.Session(), req.ResID3(), req.Body(), []modulebase.ManagerContext{{req.Mod1(), req.ResID()}, {req.Mod2(), req.ResID2()}})
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

func (f *ResourceHandlers) patchInContextsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2().WithMod3()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	module := req.Mod1()
	module2 := req.Mod2()
	module3 := req.Mod3()
	session := req.Session()
	body := req.Body()

	obj, e := module3.PatchInContexts(session, req.ResID3(), body, []modulebase.ManagerContext{{module, req.ResID()}, {module2, req.ResID2()}})
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

// batchDelete
// DELETE /<resname>
func (f *ResourceHandlers) batchDeleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}

	idlist := fetchIdList(req.Query(), w)
	if idlist == nil {
		return
	}
	ret := req.Mod1().BatchDeleteWithParam(req.Session(), idlist, req.Query(), req.Body())
	w.WriteHeader(207)
	appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
}

// delete single
// DELETE /<resname>/<resid>
func (f *ResourceHandlers) deleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	obj, e := req.Mod1().DeleteWithParam(req.Session(), req.ResID(), req.Query(), req.Body())
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

// batch detach
// /<resname>/<resid>/<resname2>
func (f *ResourceHandlers) batchDetachHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	idlist := fetchIdList(req.Query(), w)
	if idlist == nil {
		return
	}
	session := req.Session()
	module := req.Mod1()
	module2 := req.Mod2()
	jmod, e := modulebase.GetJointModule2(session, module, module2)
	var ret []modulebase.SubmitResult
	if e == nil {
		ret = jmod.BatchDetach(session, req.ResID(), idlist)
	} else {
		jmod, e := modulebase.GetJointModule2(session, module2, module)
		if e == nil {
			ret = jmod.BatchDetach2(session, req.ResID(), idlist)
		} else {
			ret = module2.BatchDeleteInContextWithParam(session, idlist, req.Query(), req.Body(), module, req.ResID())
		}
	}
	w.WriteHeader(207)
	appsrv.SendJSON(w, modulebase.SubmitResults2JSON(ret))
}

// detach joint
// delete in context
// DELETE /<resname>/<resid>/<resname2>/<resid2>
func (f *ResourceHandlers) detachHandle(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	session := req.Session()
	module := req.Mod1()
	module2 := req.Mod2()

	jmod, e := modulebase.GetJointModule2(session, module, module2)
	var obj jsonutils.JSONObject
	if e == nil { // joint detach
		obj, e = jmod.Detach(session, req.ResID(), req.ResID2(), nil)
	} else {
		obj, e = module2.DeleteInContextWithParam(session, req.ResID2(), req.Query(), req.Body(), module, req.ResID())
	}
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}

// delete in contexts
// DELETE /<resname>/<resid>/<resname2>/<resid2>/<resname3>/<resid3>
func (f *ResourceHandlers) deleteInContextsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := newRequest(ctx, w, r).WithMod1().WithMod2().WithMod3()
	if err := req.Error(); err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	session := req.Session()
	module := req.Mod1()
	module2 := req.Mod2()
	module3 := req.Mod3()
	query := req.Query()
	body := req.Body()

	obj, e := module3.DeleteInContextsWithParam(session, req.ResID3(), query, body, []modulebase.ManagerContext{{module, req.ResID()}, {module2, req.ResID2()}})
	if e != nil {
		httperrors.GeneralServerError(w, e)
	} else {
		appsrv.SendJSON(w, obj)
	}
}
