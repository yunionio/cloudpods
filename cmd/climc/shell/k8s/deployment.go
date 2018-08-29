package k8s

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func initDeployment() {
	cmdN := func(suffix string) string {
		return resourceCmdN("deployment", suffix)
	}

	type listOpt struct {
		namespaceListOptions
		baseListOptions
	}
	R(&listOpt{}, cmdN("list"), "List k8s deployment", func(s *mcclient.ClientSession, args *listOpt) error {
		params := fetchNamespaceParams(args.namespaceListOptions)
		params.Update(fetchPagingParams(args.baseListOptions))
		ret, err := k8s.Deployments.ListInContexts(s, params, args.ClusterContext())
		if err != nil {
			return err
		}
		printList(ret, k8s.Deployments.GetColumns(s))
		return nil
	})

	type createOpt struct {
		namespaceOptions
		NAME            string   `help:"Name of deployment"`
		Image           string   `help:"The image for the container to run"`
		Replicas        int64    `help:"Number of replicas for pods in this deployment"`
		RunAsPrivileged bool     `help:"Whether to run the container as privileged user"`
		Labels          string   `help:"Comma separated labels to apply to the pod(s)"`
		Env             []string `help:"Environment variables to set in container"`
		Port            []string `help:"Port for the service that is created, format is <protocol>:<service_port>:<container_port> e.g. tcp:80:3000"`
		Net             string   `help:"Network config, e.g. net1, net1:10.168.222.171"`
	}
	R(&createOpt{}, cmdN("create"), "Create deployment resource", func(s *mcclient.ClientSession, args *createOpt) error {
		params := jsonutils.NewDict()
		if len(args.Image) == 0 {
			return fmt.Errorf("Image must provided")
		}
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.Image), "containerImage")
		if args.Namespace != "" {
			params.Add(jsonutils.NewString(args.Namespace), "namespace")
		}
		if args.Replicas > 1 {
			params.Add(jsonutils.NewInt(args.Replicas), "replicas")
		}
		if args.RunAsPrivileged {
			params.Add(jsonutils.JSONTrue, "runAsPrivileged")
		}
		if len(args.Port) != 0 {
			portMappings, err := parsePortMappings(args.Port)
			if err != nil {
				return err
			}
			params.Add(portMappings, "portMappings")
		}
		if args.Net != "" {
			net, err := parseNetConfig(args.Net)
			if err != nil {
				return err
			}
			params.Add(net, "networkConfig")
		}
		ret, err := k8s.Deployments.CreateInContexts(s, params, args.ClusterContext())
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	type getOpt struct {
		resourceGetOptions
	}
	R(&getOpt{}, cmdN("show"), "Get deployment details", func(s *mcclient.ClientSession, args *getOpt) error {
		id := args.NAME
		params := jsonutils.NewDict()
		if args.Namespace != "" {
			params.Add(jsonutils.NewString(args.Namespace), "namespace")
		}
		ret, err := k8s.Deployments.GetInContexts(s, id, params, args.ClusterContext())
		if err != nil {
			return err
		}
		printObjectYAML(ret)
		return nil
	})
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
