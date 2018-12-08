package modules

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type BaseManager struct {
	serviceType  string
	endpointType string
	version      string
	apiVersion   string

	columns      []string
	adminColumns []string
}

func NewBaseManager(serviceType, endpointType, version string, columns, adminColumns []string) *BaseManager {
	return &BaseManager{
		serviceType:  serviceType,
		endpointType: endpointType,
		version:      version,
		columns:      columns,
		adminColumns: adminColumns,
	}
}

func (this *BaseManager) GetColumns(session *mcclient.ClientSession) []string {
	cols := this.columns
	if session.HasSystemAdminPrivelege() && len(this.adminColumns) > 0 {
		cols = append(cols, this.adminColumns...)
	}
	return cols
}

func (this *BaseManager) SetVersion(v string) {
	this.version = v
}

func (this *BaseManager) SetApiVersion(v string) {
	this.apiVersion = v
}

func (this *BaseManager) GetApiVersion() string {
	return this.apiVersion
}

func (this *BaseManager) versionedURL(path string) string {
	offset := 0
	for ; path[offset] == '/'; offset++ {
	}
	var ret string
	if len(this.version) > 0 {
		ret = fmt.Sprintf("/%s/%s", this.version, path[offset:])
	} else {
		ret = fmt.Sprintf("/%s", path[offset:])
	}
	// log.Debugf("versionedURL %s %s => %s", this.version, path, ret)
	return ret
}

func (this *BaseManager) jsonRequest(session *mcclient.ClientSession,
	method string, path string,
	header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return session.JSONVersionRequest(this.serviceType, this.endpointType,
		method, this.versionedURL(path),
		header, body, this.GetApiVersion())
}

func (this *BaseManager) rawRequest(session *mcclient.ClientSession,
	method string, path string,
	header http.Header, body io.Reader) (*http.Response, error) {
	return session.RawVersionRequest(this.serviceType, this.endpointType,
		method, this.versionedURL(path),
		header, body, this.GetApiVersion())
}

type ListResult struct {
	Data   []jsonutils.JSONObject
	Total  int
	Limit  int
	Offset int
}

func ListResult2JSONWithKey(result *ListResult, key string) jsonutils.JSONObject {
	obj := jsonutils.NewDict()
	if result.Total > 0 {
		obj.Add(jsonutils.NewInt(int64(result.Total)), "total")
	}
	if result.Limit > 0 {
		obj.Add(jsonutils.NewInt(int64(result.Limit)), "limit")
	}
	if result.Offset > 0 {
		obj.Add(jsonutils.NewInt(int64(result.Offset)), "offset")
	}
	arr := jsonutils.NewArray(result.Data...)
	obj.Add(arr, key)
	return obj
}

func ListResult2JSON(result *ListResult) jsonutils.JSONObject {
	return ListResult2JSONWithKey(result, "data")
}

func JSON2ListResult(result jsonutils.JSONObject) *ListResult {
	total, _ := result.Int("total")
	limit, _ := result.Int("limit")
	offset, _ := result.Int("offset")
	data, _ := result.GetArray("data")
	return &ListResult{Data: data, Total: int(total), Limit: int(limit), Offset: int(offset)}
}

func (this *BaseManager) _list(session *mcclient.ClientSession, path, responseKey string) (*ListResult, error) {
	_, body, err := this.jsonRequest(session, "GET", path, nil, nil)
	// log.Debugf("%#v %#v %#v", body, err, responseKey)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("empty response")
	}
	rets, err := body.GetArray(responseKey)
	if err != nil {
		return nil, err
	}
	total, err := body.Int("total")
	if err != nil {
		total = int64(len(rets))
	}
	if total == 0 {
		total = int64(len(rets))
	}
	limit, err := body.Int("limit")
	if err != nil {
		limit = 0
	}
	offset, err := body.Int("offset")
	if err != nil {
		offset = 0
	}
	return &ListResult{rets, int(total), int(limit), int(offset)}, nil
}

func (this *BaseManager) _submit(session *mcclient.ClientSession, method string, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
	hdr, resp, e := this.jsonRequest(session, method, path, nil, body)
	if e != nil {
		return nil, e
	}
	if method == "HEAD" {
		ret := jsonutils.NewDict()
		hdrPrefix := fmt.Sprintf("x-%s-", respKey)
		for k, v := range hdr {
			k = strings.ToLower(k)
			if strings.HasPrefix(k, hdrPrefix) && len(v) > 0 {
				if len(v) == 1 {
					ret.Add(jsonutils.NewString(v[0]), k)
				} else {
					ret.Add(jsonutils.NewStringArray(v), k)
				}
			}
		}
		return ret, nil
	}
	if resp == nil { // no reslt
		return jsonutils.NewDict(), nil
	}
	if len(respKey) == 0 {
		return resp, nil
	}
	ret, e := resp.Get(respKey)
	if e != nil {
		return nil, e
	}
	return ret, nil
}

type SubmitResult struct {
	Status int
	Id     string
	Data   jsonutils.JSONObject
}

func SubmitResults2JSON(results []SubmitResult) jsonutils.JSONObject {
	arr := jsonutils.NewArray()
	for _, r := range results {
		obj := jsonutils.NewDict()
		obj.Add(jsonutils.NewInt(int64(r.Status)), "status")
		obj.Add(jsonutils.NewString(r.Id), "id")
		obj.Add(r.Data, "data")
		arr.Add(obj)
	}
	body := jsonutils.NewDict()
	body.Add(arr, "data")
	return body
}

func SubmitResults2ListResult(results []SubmitResult) *ListResult {
	arr := make([]jsonutils.JSONObject, 0)
	for _, r := range results {
		if r.Status == 200 {
			arr = append(arr, r.Data)
		}
	}
	return &ListResult{Data: arr, Total: len(arr), Limit: 0, Offset: 0}
}

func (this *BaseManager) _batch(session *mcclient.ClientSession, method string, path string, ids []string, body jsonutils.JSONObject, respKey string) []SubmitResult {
	return BatchDo(ids, func(id string) (jsonutils.JSONObject, error) {
		u := fmt.Sprintf(path, url.PathEscape(id))
		return this._submit(session, method, u, body, respKey)
	})
}

func BatchDo(ids []string, do func(id string) (jsonutils.JSONObject, error)) []SubmitResult {
	results := make(chan SubmitResult, len(ids))
	for _, id := range ids {
		go func(id string) {
			r, e := do(id)
			if e != nil {
				ecls, ok := e.(*httputils.JSONClientError)
				if ok {
					results <- SubmitResult{Status: ecls.Code, Id: id, Data: jsonutils.NewString(ecls.Details)}
				} else {
					results <- SubmitResult{Status: 400, Id: id, Data: jsonutils.NewString(e.Error())}
				}
			} else {
				results <- SubmitResult{Status: 200, Id: id, Data: r}
			}
		}(id)
	}
	ret := make([]SubmitResult, len(ids))
	for i := 0; i < len(ids); i++ {
		ret[i] = <-results
	}
	close(results)
	return ret
}

func (this *BaseManager) _get(session *mcclient.ClientSession, path string, respKey string) (jsonutils.JSONObject, error) {
	/* _, body, err := this.jsonRequest(session, "GET", path, nil, nil)
	   if err != nil {
	       return nil, err
	   }
	   con, err := body.Get(responseKey)
	   if err != nil {
	       return nil, err
	   }
	   return con, nil */
	return this._submit(session, "GET", path, nil, respKey)
}

func (this *BaseManager) _head(session *mcclient.ClientSession, path string, respKey string) (jsonutils.JSONObject, error) {
	return this._submit(session, "HEAD", path, nil, respKey)
}

func (this *BaseManager) _post(session *mcclient.ClientSession, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
	return this._submit(session, "POST", path, body, respKey)
}

func (this *BaseManager) _put(session *mcclient.ClientSession, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
	return this._submit(session, "PUT", path, body, respKey)
}

func (this *BaseManager) _patch(session *mcclient.ClientSession, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
	return this._submit(session, "PATCH", path, body, respKey)
}

func (this *BaseManager) _delete(session *mcclient.ClientSession, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
	return this._submit(session, "DELETE", path, body, respKey)
}
