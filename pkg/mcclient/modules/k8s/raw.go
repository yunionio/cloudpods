package k8s

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
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
	query jsonutils.JSONObject
}

func newRawResource(kind, namespace, name string, query jsonutils.JSONObject) *rawResource {
	nsQuery := getNamespaceQuery(namespace)
	if query != nil {
		nsQuery.Update(query)
	}
	ctx := &rawResource{kind: kind, name: name, query: nsQuery}
	return ctx
}

func (ctx rawResource) path() string {
	segs := make([]string, 0)
	segs = append(segs, "_raw", ctx.kind, ctx.name)
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

func (m *RawResourceManager) request(s *mcclient.ClientSession, method string, path string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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

func (m *RawResourceManager) Get(s *mcclient.ClientSession, kind string, namespace string, name string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ctx := newRawResource(kind, namespace, name, query)
	return m.request(s, "GET", ctx.path(), nil)
}

func (m *RawResourceManager) Put(s *mcclient.ClientSession, kind string, namespace string, name string, body jsonutils.JSONObject) error {
	rawBytes := body.String()
	newBody := jsonutils.NewDict()
	newBody.Add(jsonutils.NewString(rawBytes), "raw")
	ctx := newRawResource(kind, namespace, name, nil)
	_, err := m.request(s, "PUT", ctx.path(), newBody)
	return err
}

func (m *RawResourceManager) Delete(s *mcclient.ClientSession, kind string, namespace string, name string, query jsonutils.JSONObject) error {
	ctx := newRawResource(kind, namespace, name, query)
	_, err := m.request(s, "DELETE", ctx.path(), nil)
	return err
}
