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

package k8s

import (
	"fmt"
	"io"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

var RawResource *RawResourceManager

func init() {
	RawResource = &RawResourceManager{
		serviceType: "k8s",
	}
}

type rawResource struct {
	kind  string
	name  string
	segs  []string
	query jsonutils.JSONObject
}

func newRawResource(kind, namespace, name, cluster string) *rawResource {
	nsQuery := getNamespaceQuery(namespace)
	if cluster != "" {
		nsQuery.Add(jsonutils.NewString(cluster), "cluster")
	}
	ctx := &rawResource{kind: kind, name: name, segs: make([]string, 0), query: nsQuery}
	return ctx
}

func (ctx *rawResource) addSegs(seg ...string) {
	ctx.segs = append(ctx.segs, seg...)
}

func (ctx rawResource) path() string {
	segs := make([]string, 0)
	segs = append(segs, "_raw", ctx.kind, ctx.name)
	segs = append(segs, ctx.segs...)
	path := fmt.Sprintf("/%s", strings.Join(segs, "/"))
	if ctx.query != nil {
		qs := ctx.query.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return path
}

type RawResourceManager struct {
	serviceType string
}

func (m *RawResourceManager) request(s *mcclient.ClientSession, method httputils.THttpMethod, path string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	_, ret, err := s.JSONRequest(m.serviceType, "", method, path, nil, body)
	return ret, err
}

func getNamespaceQuery(namespace string) *jsonutils.JSONDict {
	query := jsonutils.NewDict()
	if namespace != "" {
		query.Set("namespace", jsonutils.NewString(namespace))
	}
	return query
}

func (m *RawResourceManager) Get(s *mcclient.ClientSession, kind string, namespace string, name string, cluster string) (jsonutils.JSONObject, error) {
	ctx := newRawResource(kind, namespace, name, cluster)
	return m.request(s, "GET", ctx.path(), nil)
}

func (m *RawResourceManager) GetYAML(s *mcclient.ClientSession, kind string, namespace string, name string, cluster string) ([]byte, error) {
	ctx := newRawResource(kind, namespace, name, cluster)
	ctx.addSegs("yaml")
	resp, err := s.RawRequest(m.serviceType, "", "GET", ctx.path(), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (m *RawResourceManager) Put(s *mcclient.ClientSession, kind string, namespace string, name string, body jsonutils.JSONObject, cluster string) error {
	ctx := newRawResource(kind, namespace, name, cluster)
	_, err := m.request(s, "PUT", ctx.path(), body)
	return err
}

func (m *RawResourceManager) Delete(s *mcclient.ClientSession, kind string, namespace string, name string, cluster string) error {
	ctx := newRawResource(kind, namespace, name, cluster)
	_, err := m.request(s, "DELETE", ctx.path(), nil)
	return err
}
