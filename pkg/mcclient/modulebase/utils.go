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
	"io"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func Get(manager ResourceManager, session *mcclient.ClientSession, path string, respKey string) (jsonutils.JSONObject, error) {
	return manager._get(session, path, respKey)
}

func List(manager ResourceManager, session *mcclient.ClientSession, path, respKey string) (*printutils.ListResult, error) {
	return manager._list(session, path, respKey)
}

func Head(manager ResourceManager, session *mcclient.ClientSession, path string, respKey string) (jsonutils.JSONObject, error) {
	return manager._head(session, path, respKey)
}

func Post(manager ResourceManager, session *mcclient.ClientSession, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
	return manager._post(session, path, body, respKey)
}

func Put(manager ResourceManager, session *mcclient.ClientSession, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
	return manager._put(session, path, body, respKey)
}

func Patch(manager ResourceManager, session *mcclient.ClientSession, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
	return manager._patch(session, path, body, respKey)
}

func Delete(manager ResourceManager, session *mcclient.ClientSession, path string, body jsonutils.JSONObject, respKey string) (jsonutils.JSONObject, error) {
	return manager._delete(session, path, body, respKey)
}

func RawRequest(manager ResourceManager, session *mcclient.ClientSession,
	method httputils.THttpMethod, path string,
	header http.Header, body io.Reader) (*http.Response, error) {
	return session.RawVersionRequest(manager.serviceType, manager.endpointType,
		method, manager.versionedURL(path),
		header, body)
}

func JsonRequest(manager ResourceManager, session *mcclient.ClientSession,
	method httputils.THttpMethod, path string,
	header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return session.JSONVersionRequest(manager.serviceType, manager.endpointType,
		method, manager.versionedURL(path),
		header, body)
}
