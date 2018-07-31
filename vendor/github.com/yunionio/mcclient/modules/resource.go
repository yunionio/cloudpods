package modules

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/pkg/httperrors"
	"github.com/yunionio/pkg/util/httputils"

	"github.com/yunionio/mcclient"
)

type ResourceManager struct {
	BaseManager
	context       string
	Keyword       string
	KeywordPlural string
}

func (this *ResourceManager) KeyString() string {
	return this.KeywordPlural
}

func (this *ResourceManager) Version() string {
	return this.version
}

func (this *ResourceManager) ServiceType() string {
	return this.serviceType
}

func (this *ResourceManager) EndpointType() string {
	return this.endpointType
}

func (this *ResourceManager) URLPath() string {
	return strings.Replace(this.KeywordPlural, ":", "/", -1)
}

func (this *ResourceManager) ContextPath(ctxs []ManagerContext) string {
	segs := make([]string, 0)
	if len(this.context) > 0 {
		segs = append(segs, this.context)
	}
	if ctxs != nil && len(ctxs) > 0 {
		for _, ctx := range ctxs {
			segs = append(segs, ctx.InstanceManager.KeyString())
			if len(ctx.InstanceId) > 0 {
				segs = append(segs, url.PathEscape(ctx.InstanceId))
			}
		}
	}
	segs = append(segs, this.URLPath())
	return strings.Join(segs, "/")
}

func (this *ResourceManager) GetById(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.GetByIdInContexts(session, id, params, nil)
}

func (this *ResourceManager) GetByIdInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.GetByIdInContexts(session, id, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) GetByIdInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s", this.ContextPath(ctxs), url.PathEscape(id))
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return this._get(session, path, this.Keyword)
}

func (this *ResourceManager) GetByName(session *mcclient.ClientSession, name string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.GetByNameInContexts(session, name, params, nil)
}

func (this *ResourceManager) GetByNameInContext(session *mcclient.ClientSession, name string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.GetByNameInContexts(session, name, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) GetByNameInContexts(session *mcclient.ClientSession, name string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	var paramsDict *jsonutils.JSONDict
	if params != nil {
		paramsDict = params.(*jsonutils.JSONDict)
		paramsDict = paramsDict.Copy()
	} else {
		paramsDict = jsonutils.NewDict()
	}
	paramsDict.Add(jsonutils.NewString(name), "name")
	results, e := this.ListInContexts(session, paramsDict, ctxs)
	if e != nil {
		return nil, e
	}
	if len(results.Data) == 0 {
		return nil, httperrors.NewNotFoundError("Name %s not found", name)
	} else if len(results.Data) == 1 {
		return results.Data[0], nil
	} else {
		return nil, httperrors.NewDuplicateNameError("Name %s duplicate", name)
	}
}

func (this *ResourceManager) Get(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.GetInContexts(session, id, params, nil)
}

func (this *ResourceManager) GetInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.GetInContexts(session, id, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) GetInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	obj, e := this.GetByIdInContexts(session, id, params, ctxs)
	if e != nil {
		je, ok := e.(*httputils.JSONClientError)
		if ok && je.Code == 404 {
			return this.GetByNameInContexts(session, id, params, ctxs)
		} else {
			return nil, e
		}
	} else {
		return obj, e
	}
}

func (this *ResourceManager) GetId(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (string, error) {
	return this.GetIdInContexts(session, id, params, nil)
}

func (this *ResourceManager) GetIdInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (string, error) {
	return this.GetIdInContexts(session, id, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) GetIdInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (string, error) {
	obj, e := this.Get(session, id, params)
	if e != nil {
		return "", e
	}
	return obj.GetString("id")
}

func (this *ResourceManager) GetSpecific(session *mcclient.ClientSession, id string, spec string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.GetSpecificInContexts(session, id, spec, params, nil)
}

func (this *ResourceManager) GetSpecificInContext(session *mcclient.ClientSession, id string, spec string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.GetSpecificInContexts(session, id, spec, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) GetSpecificInContexts(session *mcclient.ClientSession, id string, spec string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s", this.ContextPath(ctxs), url.PathEscape(id), url.PathEscape(spec))
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return this._get(session, path, this.Keyword)
}

func (this *ResourceManager) BatchGet(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []SubmitResult {
	return this.BatchGetInContexts(session, idlist, params, nil)
}

func (this *ResourceManager) BatchGetInContext(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult {
	return this.BatchGetInContexts(session, idlist, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) BatchGetInContexts(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult {
	return BatchDo(idlist, func(id string) (jsonutils.JSONObject, error) {
		return this.GetInContexts(session, id, params, ctxs)
	})
}

func (this *ResourceManager) List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*ListResult, error) {
	return this.ListInContexts(session, params, nil)
}

func (this *ResourceManager) ListInContext(session *mcclient.ClientSession, params jsonutils.JSONObject, ctx Manager, ctxid string) (*ListResult, error) {
	return this.ListInContexts(session, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) ListInContexts(session *mcclient.ClientSession, params jsonutils.JSONObject, ctxs []ManagerContext) (*ListResult, error) {
	path := fmt.Sprintf("/%s", this.ContextPath(ctxs))
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return this._list(session, path, this.KeywordPlural)
}

func (this *ResourceManager) Head(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.HeadInContexts(session, id, params, nil)
}

func (this *ResourceManager) HeadInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.HeadInContexts(session, id, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) HeadInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s", this.ContextPath(ctxs), url.PathEscape(id))
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return this._head(session, path, this.Keyword)
}

func (this *ResourceManager) params2Body(params jsonutils.JSONObject) *jsonutils.JSONDict {
	body := jsonutils.NewDict()
	if params != nil {
		body.Add(params, this.Keyword)
	}
	return body
}

func (this *ResourceManager) Create(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.CreateInContexts(session, params, nil)
}

func (this *ResourceManager) CreateInContext(session *mcclient.ClientSession, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.CreateInContexts(session, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) CreateInContexts(session *mcclient.ClientSession, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s", this.ContextPath(ctxs))
	return this._post(session, path, this.params2Body(params), this.Keyword)
}

func (this *ResourceManager) BatchCreate(session *mcclient.ClientSession, params jsonutils.JSONObject, count int) []SubmitResult {
	return this.BatchCreateInContexts(session, params, count, nil)
}

func (this *ResourceManager) BatchCreateInContext(session *mcclient.ClientSession, params jsonutils.JSONObject, count int, ctx Manager, ctxid string) []SubmitResult {
	return this.BatchCreateInContexts(session, params, count, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) BatchCreateInContexts(session *mcclient.ClientSession, params jsonutils.JSONObject, count int, ctxs []ManagerContext) []SubmitResult {
	path := fmt.Sprintf("/%s", this.ContextPath(ctxs))
	body := this.params2Body(params)
	body.Add(jsonutils.NewInt(int64(count)), "count")
	ret := make([]SubmitResult, count)
	respbody, err := this._post(session, path, body, this.KeywordPlural)
	if err != nil {
		for i := 0; i < count; i++ {
			ret[i] = SubmitResult{Status: 500, Data: jsonutils.NewString(err.Error())}
		}
		return ret
	}
	respArray, ok := respbody.(*jsonutils.JSONArray)
	if !ok {
		for i := 0; i < count; i++ {
			ret[i] = SubmitResult{Status: 500, Data: jsonutils.NewString("Invalid response")}
		}
		return ret
	}
	for i := 0; i < respArray.Size(); i++ {
		json, e := respArray.GetAt(i)
		if e != nil {
			ret[i] = SubmitResult{Status: 500, Data: jsonutils.NewString(e.Error())}
		} else {
			code, _ := json.Int("status")
			dat, _ := json.Get("body")
			idstr, _ := json.GetString("id")
			ret[i] = SubmitResult{Status: int(code), Id: idstr, Data: dat}
		}
	}
	return ret
}

func (this *ResourceManager) Update(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.PutInContexts(session, id, params, nil)
}

func (this *ResourceManager) Put(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.PutInContexts(session, id, params, nil)
}

func (this *ResourceManager) PutInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.PutInContexts(session, id, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) PutInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s", this.ContextPath(ctxs), url.PathEscape(id))
	return this._put(session, path, this.params2Body(params), this.Keyword)
}

func (this *ResourceManager) BatchUpdate(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []SubmitResult {
	return this.BatchPutInContexts(session, idlist, params, nil)
}

func (this *ResourceManager) BatchPut(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []SubmitResult {
	return this.BatchPutInContexts(session, idlist, params, nil)
}

func (this *ResourceManager) BatchPutInContext(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult {
	return this.BatchPutInContexts(session, idlist, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) BatchPutInContexts(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult {
	return BatchDo(idlist, func(id string) (jsonutils.JSONObject, error) {
		return this.PutInContexts(session, id, params, ctxs)
	})
}

func (this *ResourceManager) Patch(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.PatchInContexts(session, id, params, nil)
}

func (this *ResourceManager) PatchInContext(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.PatchInContexts(session, id, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) PatchInContexts(session *mcclient.ClientSession, id string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s", this.ContextPath(ctxs), url.PathEscape(id))
	return this._patch(session, path, this.params2Body(params), this.Keyword)
}

func (this *ResourceManager) BatchPatch(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []SubmitResult {
	return this.BatchPatchInContexts(session, idlist, params, nil)
}

func (this *ResourceManager) BatchPatchInContext(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult {
	return this.BatchPatchInContexts(session, idlist, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) BatchPatchInContexts(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult {
	return BatchDo(idlist, func(id string) (jsonutils.JSONObject, error) {
		return this.PatchInContexts(session, id, params, ctxs)
	})
}

func (this *ResourceManager) PerformAction(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.PerformActionInContexts(session, id, action, params, nil)
}

func (this *ResourceManager) PerformActionInContext(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.PerformActionInContexts(session, id, action, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) PerformActionInContexts(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s", this.ContextPath(ctxs), url.PathEscape(id), url.PathEscape(action))
	return this._post(session, path, this.params2Body(params), this.Keyword)
}

func (this *ResourceManager) BatchPerformAction(session *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject) []SubmitResult {
	return this.BatchPerformActionInContexts(session, idlist, action, params, nil)
}

func (this *ResourceManager) BatchPerformActionInContext(session *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult {
	return this.BatchPerformActionInContexts(session, idlist, action, params, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) BatchPerformActionInContexts(session *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult {
	return BatchDo(idlist, func(id string) (jsonutils.JSONObject, error) {
		return this.PerformActionInContexts(session, id, action, params, ctxs)
	})
}

func (this *ResourceManager) Delete(session *mcclient.ClientSession, id string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.DeleteInContexts(session, id, body, nil)
}

func (this *ResourceManager) DeleteWithParam(session *mcclient.ClientSession, id string, params, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.DeleteInContextsWithParam(session, id, params, body, nil)
}

func (this *ResourceManager) DeleteInContext(session *mcclient.ClientSession, id string, body jsonutils.JSONObject, ctx Manager, ctxid string) (jsonutils.JSONObject, error) {
	return this.DeleteInContexts(session, id, body, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) DeleteInContexts(session *mcclient.ClientSession, id string, body jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	return this.deleteInContexts(session, id, nil, body, ctxs)
}

func (this *ResourceManager) DeleteInContextsWithParam(session *mcclient.ClientSession, id string, params, body jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	return this.deleteInContexts(session, id, params, body, ctxs)
}

func (this *ResourceManager) deleteInContexts(session *mcclient.ClientSession, id string, params, body jsonutils.JSONObject, ctxs []ManagerContext) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s", this.ContextPath(ctxs), url.PathEscape(id))
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	if body != nil {
		body = this.params2Body(body)
	}
	return this._delete(session, path, body, this.Keyword)
}

func (this *ResourceManager) BatchDelete(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject) []SubmitResult {
	return this.BatchDeleteInContexts(session, idlist, body, nil)
}

func (this *ResourceManager) BatchDeleteWithParam(session *mcclient.ClientSession, idlist []string, params, body jsonutils.JSONObject) []SubmitResult {
	return this.BatchDeleteInContextsWithParam(session, idlist, params, body, nil)
}

func (this *ResourceManager) BatchDeleteInContext(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult {
	return this.BatchDeleteInContexts(session, idlist, body, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) BatchDeleteInContextWithParam(session *mcclient.ClientSession, idlist []string, params, body jsonutils.JSONObject, ctx Manager, ctxid string) []SubmitResult {
	return this.BatchDeleteInContextsWithParam(session, idlist, params, body, []ManagerContext{{ctx, ctxid}})
}

func (this *ResourceManager) BatchDeleteInContexts(session *mcclient.ClientSession, idlist []string, body jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult {
	return BatchDo(idlist, func(id string) (jsonutils.JSONObject, error) {
		return this.DeleteInContexts(session, id, body, ctxs)
	})
}

func (this *ResourceManager) BatchDeleteInContextsWithParam(session *mcclient.ClientSession, idlist []string, params, body jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult {
	return BatchDo(idlist, func(id string) (jsonutils.JSONObject, error) {
		return this.deleteInContexts(session, id, params, body, ctxs)
	})
}

func (this *ResourceManager) GetMetadata(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this.GetSpecific(session, id, "metadata", params)
}
