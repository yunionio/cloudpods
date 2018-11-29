package k8s

import (
	"yunion.io/x/jsonutils"
)

type IngressCreateOptions struct {
	NamespaceWithClusterOptions
	NAME    string `help:"Name of ingress"`
	SERVICE string `help:"Service name"`
	PORT    int    `help:"Service port"`
	Path    string `help:"HTTP path"`
	Host    string `help:"Fuly qualified domain name of a network host" required:"true"`
}

func (o IngressCreateOptions) Params() *jsonutils.JSONDict {
	params := o.NamespaceWithClusterOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	path := jsonutils.NewDict()
	path.Add(jsonutils.NewString(o.SERVICE), "backend", "serviceName")
	path.Add(jsonutils.NewInt(int64(o.PORT)), "backend", "servicePort")
	if o.Path != "" {
		path.Add(jsonutils.NewString(o.Path), "path")
	}
	paths := jsonutils.NewArray()
	paths.Add(path)
	rule := jsonutils.NewDict()
	rule.Add(paths, "http", "paths")
	rule.Add(jsonutils.NewString(o.Host), "host")
	rules := jsonutils.NewArray()
	rules.Add(rule)
	params.Add(rules, "rules")
	return params
}
