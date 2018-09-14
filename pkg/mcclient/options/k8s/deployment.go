package k8s

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
)

type DeploymentCreateOptions struct {
	NamespaceWithClusterOptions
	NAME            string   `help:"Name of deployment"`
	Image           string   `help:"The image for the container to run"`
	Replicas        int64    `help:"Number of replicas for pods in this deployment"`
	RunAsPrivileged bool     `help:"Whether to run the container as privileged user"`
	Labels          string   `help:"Comma separated labels to apply to the pod(s), e.g. --labels='app=hazelcast,env=prod'"`
	Env             []string `help:"Environment variables to set in container"`
	Port            []string `help:"Port for the service that is created, format is <protocol>:<service_port>:<container_port> e.g. tcp:80:3000"`
	Net             string   `help:"Network config, e.g. net1, net1:10.168.222.171"`
}

func (o DeploymentCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceWithClusterOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	if len(o.Image) == 0 {
		return nil, fmt.Errorf("Image must provided")
	}
	params.Add(jsonutils.NewString(o.Image), "containerImage")
	if o.Replicas > 1 {
		params.Add(jsonutils.NewInt(o.Replicas), "replicas")
	}
	if o.RunAsPrivileged {
		params.Add(jsonutils.JSONTrue, "runAsPrivileged")
	}
	if len(o.Port) != 0 {
		portMappings, err := parsePortMappings(o.Port)
		if err != nil {
			return nil, err
		}
		params.Add(portMappings, "portMappings")
	}
	envList := jsonutils.NewArray()
	for _, env := range o.Env {
		parts := strings.Split(env, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Bad env value: %v", env)
		}
		envObj := jsonutils.NewDict()
		envObj.Add(jsonutils.NewString(parts[0]), "name")
		envObj.Add(jsonutils.NewString(parts[1]), "value")
		envList.Add(envObj)
	}
	params.Add(envList, "variables")
	if o.Net != "" {
		net, err := parseNetConfig(o.Net)
		if err != nil {
			return nil, err
		}
		params.Add(net, "networkConfig")
	}
	if o.Labels != "" {
		labels, err := parseLabels(o.Labels)
		if err != nil {
			return nil, err
		}
		params.Add(labels, "labels")
	}
	return params, nil
}

type portMapping struct {
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	Protocol   string `json:"protocol"`
}

func parsePortMapping(port string) (*portMapping, error) {
	if len(port) == 0 {
		return nil, fmt.Errorf("empty port mapping desc string")
	}
	parts := strings.Split(port, ":")
	mapping := &portMapping{}
	for _, part := range parts {
		if sets.NewString("tcp", "udp").Has(strings.ToLower(part)) {
			mapping.Protocol = strings.ToUpper(part)
		}
		if port, err := strconv.Atoi(part); err != nil {
			continue
		} else {
			if mapping.Port == 0 {
				mapping.Port = int32(port)
			} else {
				mapping.TargetPort = int32(port)
			}
		}
	}
	if mapping.Protocol == "" {
		mapping.Protocol = "TCP"
	}
	if mapping.Port <= 0 {
		return nil, fmt.Errorf("Service port not provided")
	}
	if mapping.TargetPort < 0 {
		return nil, fmt.Errorf("Container invalid targetPort %d", mapping.TargetPort)
	}
	if mapping.TargetPort == 0 {
		mapping.TargetPort = mapping.Port
	}
	return mapping, nil
}

func parsePortMappings(ports []string) (*jsonutils.JSONArray, error) {
	ret := jsonutils.NewArray()
	for _, port := range ports {
		mapping, err := parsePortMapping(port)
		if err != nil {
			return nil, fmt.Errorf("Port %q error: %v", port, err)
		}
		ret.Add(jsonutils.Marshal(mapping))
	}
	return ret, nil
}

func parseNetConfig(net string) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()
	for _, p := range strings.Split(net, ":") {
		if regutils.MatchIP4Addr(p) {
			ret.Add(jsonutils.NewString(p), "address")
		} else {
			ret.Add(jsonutils.NewString(p), "network")
		}
	}
	return ret, nil
}

type DeploymentCreateFromFileOptions struct {
	NamespaceResourceGetOptions
	FILE string `help:"K8s resource YAML or JSON file"`
}

func (o DeploymentCreateFromFileOptions) Params() (*jsonutils.JSONDict, error) {
	params := o.NamespaceResourceGetOptions.Params()
	params.Add(jsonutils.NewString(o.NAME), "name")
	content, err := ioutil.ReadFile(o.FILE)
	if err != nil {
		return nil, err
	}
	params.Add(jsonutils.NewString(string(content)), "content")
	return params, nil
}

func parseLabels(labelStr string) (*jsonutils.JSONArray, error) {
	labelsStrs := strings.Split(labelStr, ",")
	ret := jsonutils.NewArray()
	if len(labelsStrs) == 0 {
		return ret, nil
	}
	for _, str := range labelsStrs {
		label, err := parseLabel(str)
		if err != nil {
			return nil, err
		}
		ret.Add(label)
	}
	return ret, nil
}

func parseLabel(str string) (jsonutils.JSONObject, error) {
	parts := strings.Split(str, "=")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid label string: %s", str)
	}
	label := jsonutils.NewDict()
	label.Add(jsonutils.NewString(parts[0]), "key")
	label.Add(jsonutils.NewString(parts[1]), "value")
	return label, nil
}
