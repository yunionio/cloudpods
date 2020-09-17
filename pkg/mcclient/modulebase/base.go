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
	if session.HasSystemAdminPrivilege() && len(this.adminColumns) > 0 {
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
	method httputils.THttpMethod, path string,
	header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return session.JSONVersionRequest(this.serviceType, this.endpointType,
		method, this.versionedURL(path),
		header, body, this.GetApiVersion())
}

func (this *BaseManager) rawRequest(session *mcclient.ClientSession,
	method httputils.THttpMethod, path string,
	header http.Header, body io.Reader) (*http.Response, error) {
	return session.RawVersionRequest(this.serviceType, this.endpointType,
		method, this.versionedURL(path),
		header, body, this.GetApiVersion())
}

func (this *BaseManager) rawBaseUrlRequest(s *mcclient.ClientSession,
	method httputils.THttpMethod, path string,
	header http.Header, body io.Reader) (*http.Response, error) {
	baseUrlF := func(baseurl string) string {
		obj, _ := url.Parse(baseurl)
		obj.Path = ""
		return obj.String()
	}
	return s.RawBaseUrlRequest(
		this.serviceType, this.endpointType,
		method, this.versionedURL(path),
		header, body, this.GetApiVersion(), baseUrlF)
}

type ListResult struct {
	Data   []jsonutils.JSONObject
	Total  int
	Limit  int
	Offset int

	NextMarker  string
	MarkerField string
	MarkerOrder string
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
	if len(result.NextMarker) > 0 {
		obj.Add(jsonutils.NewString(result.NextMarker), "next_marker")
	}
	if len(result.MarkerField) > 0 {
		obj.Add(jsonutils.NewString(result.MarkerField), "marker_field")
	}
	if len(result.MarkerOrder) > 0 {
		obj.Add(jsonutils.NewString(result.MarkerOrder), "marker_order")
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
	nextMarker, _ := result.GetString("next_marker")
	markerField, _ := result.GetString("marker_field")
	markerOrder, _ := result.GetString("marker_order")
	data, _ := result.GetArray("data")
	if len(markerField) == 0 && total == 0 {
		total = int64(len(data))
	}
	return &ListResult{
		Data:  data,
		Total: int(total), Limit: int(limit), Offset: int(offset),
		NextMarker:  nextMarker,
		MarkerField: markerField,
		MarkerOrder: markerOrder,
	}
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
	nextMarker, _ := body.GetString("next_marker")
	markerField, _ := body.GetString("marker_field")
	markerOrder, _ := body.GetString("marker_order")
	total, _ := body.Int("total")
	limit, _ := body.Int("limit")
	offset, _ := body.Int("offset")
	if len(nextMarker) == 0 && total == 0 {
		total = int64(len(rets))
	}
	return &ListResult{
		Data:  rets,
		Total: int(total), Limit: int(limit), Offset: int(offset),
		NextMarker:  nextMarker,
		MarkerField: markerField,
		MarkerOrder: markerOrder,
	}, nil
}

func (this *BaseManager) _submit(session *mcclient.ClientSession, method httputils.THttpMethod, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
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
	Id     interface{}
	Data   jsonutils.JSONObject
}

func SubmitResults2JSON(results []SubmitResult) jsonutils.JSONObject {
	arr := jsonutils.NewArray()
	for _, r := range results {
		obj := jsonutils.NewDict()
		obj.Add(jsonutils.NewInt(int64(r.Status)), "status")
		obj.Add(jsonutils.Marshal(r.Id), "id")
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

func (this *BaseManager) _batch(session *mcclient.ClientSession, method httputils.THttpMethod, path string, ids []string, body jsonutils.JSONObject, respKey string) []SubmitResult {
	return BatchDo(ids, func(id string) (jsonutils.JSONObject, error) {
		u := fmt.Sprintf(path, url.PathEscape(id))
		return this._submit(session, method, u, body, respKey)
	})
}

func addResult(results chan SubmitResult, id interface{}, r jsonutils.JSONObject, e error) {
	if e != nil {
		ecls, ok := e.(*httputils.JSONClientError)
		if ok {
			results <- SubmitResult{Status: ecls.Code, Id: id, Data: jsonutils.Marshal(ecls)}
		} else {
			results <- SubmitResult{Status: 400, Id: id, Data: jsonutils.NewString(e.Error())}
		}
	} else {
		results <- SubmitResult{Status: 200, Id: id, Data: r}
	}
}

func waitResults(results chan SubmitResult, length int) []SubmitResult {
	ret := make([]SubmitResult, length)
	for i := 0; i < length; i++ {
		ret[i] = <-results
	}
	return ret
}

func BatchDo(ids []string, do func(id string) (jsonutils.JSONObject, error)) []SubmitResult {
	results := make(chan SubmitResult, len(ids))
	for i := 0; i < len(ids); i++ {
		go func(id string) {
			r, e := do(id)
			addResult(results, id, r, e)
		}(ids[i])
	}
	return waitResults(results, len(ids))
}

func BatchParamsDo(
	ids []string, params []jsonutils.JSONObject,
	do func(id string, param jsonutils.JSONObject) (jsonutils.JSONObject, error),
) []SubmitResult {
	results := make(chan SubmitResult, len(ids))
	for i := 0; i < len(ids); i++ {
		go func(id string, param jsonutils.JSONObject) {
			r, e := do(id, param)
			addResult(results, id, r, e)
		}(ids[i], params[i])
	}
	return waitResults(results, len(ids))
}

func BatchDoClassAction(
	batchParams []jsonutils.JSONObject, do func(jsonutils.JSONObject) (jsonutils.JSONObject, error),
) []SubmitResult {
	results := make(chan SubmitResult, len(batchParams))
	for i := 0; i < len(batchParams); i++ {
		go func(params jsonutils.JSONObject) {
			r, e := do(params)
			addResult(results, params, r, e)
		}(batchParams[i])
	}
	return waitResults(results, len(batchParams))
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
