package k8s

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var RawResource *RawResourceManager

func init() {
	RawResource = &RawResourceManager{
		serviceType: "k8s",
	}
}

type rawResourceContext struct {
	kind  string
	name  string
	query jsonutils.JSONObject
	ctxs  []modules.ManagerContext
}

func newRawResourceContext(kind, namespace, name string, query jsonutils.JSONObject, ctxs []modules.ManagerContext) *rawResourceContext {
	nsQuery := getNamespaceQuery(namespace)
	if query != nil {
		nsQuery.Update(query)
	}
	ctx := &rawResourceContext{kind: kind, name: name, query: nsQuery, ctxs: ctxs}
	return ctx
}

func (ctx rawResourceContext) contextPath() string {
	segs := make([]string, 0)
	ctxs := ctx.ctxs
	if ctxs != nil && len(ctxs) > 0 {
		for _, c := range ctxs {
			segs = append(segs, c.InstanceManager.KeyString())
			if len(c.InstanceId) > 0 {
				segs = append(segs, url.PathEscape(c.InstanceId))
			}
		}
	}
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

func (m *RawResourceManager) Get(s *mcclient.ClientSession, kind string, namespace string, name string, query jsonutils.JSONObject, ctxs []modules.ManagerContext) (jsonutils.JSONObject, error) {
	ctx := newRawResourceContext(kind, namespace, name, query, ctxs)
	return m.request(s, "GET", ctx.contextPath(), nil)
}

func (m *RawResourceManager) Put(s *mcclient.ClientSession, kind string, namespace string, name string, body jsonutils.JSONObject, ctxs []modules.ManagerContext) error {
	rawBytes := body.String()
	newBody := jsonutils.NewDict()
	newBody.Add(jsonutils.NewString(rawBytes), "raw")
	ctx := newRawResourceContext(kind, namespace, name, nil, ctxs)
	_, err := m.request(s, "PUT", ctx.contextPath(), newBody)
	return err
}

func (m *RawResourceManager) Delete(s *mcclient.ClientSession, kind string, namespace string, name string, query jsonutils.JSONObject, ctxs []modules.ManagerContext) error {
	ctx := newRawResourceContext(kind, namespace, name, query, ctxs)
	_, err := m.request(s, "DELETE", ctx.contextPath(), nil)
	return err
}
