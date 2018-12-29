package k8s

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
)

type ServiceSpecOptions struct {
	Port       []string `help:"Port for the service that is created, format is <protocol>:<service_port>:<container_port> e.g. tcp:80:3000"`
	IsExternal bool     `help:"Created service is external loadbalance"`
	LbNetwork  string   `help:"LoadBalancer service network id"`
}

func (o ServiceSpecOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	if len(o.Port) == 0 {
		return params, nil
	}
	portMappings, err := parsePortMappings(o.Port)
	if err != nil {
		return nil, err
	}
	if o.IsExternal {
		params.Add(jsonutils.JSONTrue, "isExternal")
		if o.LbNetwork != "" {
			params.Add(jsonutils.NewString(o.LbNetwork), "loadBalancerNetwork")
		}
	}
	params.Add(portMappings, "portMappings")
	return params, nil
}

type ServiceCreateOptions struct {
	NamespaceWithClusterOptions
	ServiceSpecOptions
	NAME     string   `help:"Name of deployment"`
	Selector []string `help:"Selectors are backends pods labels, e.g. 'run=app'"`
}

func (o ServiceCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	svcSpec, err := o.ServiceSpecOptions.Params()
	if err != nil {
		return nil, err
	}
	selector := jsonutils.NewDict()
	for _, s := range o.Selector {
		parts := strings.Split(s, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Invalid selctor string: %s", s)
		}
		selector.Add(jsonutils.NewString(parts[1]), parts[0])
	}
	params.Update(svcSpec)
	params.Add(selector, "selector")
	params.Add(jsonutils.NewString(o.NAME), "name")
	return params, nil
}

type ServiceListOptions struct {
	NamespaceResourceListOptions
	Type string `help:"Service type" choices:"ClusterIP|LoadBalancer|NodePort|ExternalName"`
}

func (o ServiceListOptions) Params() *jsonutils.JSONDict {
	params := o.NamespaceResourceListOptions.Params()
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "type")
	}
	return params
}
