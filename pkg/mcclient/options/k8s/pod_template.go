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
	"strings"

	"yunion.io/x/jsonutils"
)

// only support one container now
type K8sPodTemplateOptions struct {
	// container option
	name            string
	Image           string   `help:"The image for the container to run" required:"true"`
	Command         string   `help:"Container start command"`
	Args            string   `help:"Container start command args"`
	Env             []string `help:"Environment variables to set in container"`
	Mem             int      `help:"Memory request MB size"`
	Cpu             float64  `help:"Cpu request cores"`
	Pvc             []string `help:"PVC volume desc, format is <pvc_name>:<mount_point>"`
	RunAsPrivileged bool     `help:"Whether to run the container as privileged user"`

	// pod option
	RestartPolicy  string   `help:"Pod restart policy" choices:"Always|OnFailure|Never"`
	RegistrySecret []string `help:"Docker registry secret"`
}

func (o *K8sPodTemplateOptions) setContainerName(name string) {
	o.name = name
}

func (o K8sPodTemplateOptions) Params() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	container := jsonutils.NewDict()
	containers := jsonutils.NewArray()

	container.Add(jsonutils.NewString(o.name), "name")
	container.Add(jsonutils.NewString(o.Image), "image")
	if len(o.Command) != 0 {
		container.Add(jsonutils.NewStringArray(strings.Split(o.Command, " ")), "command")
	}
	if len(o.Args) != 0 {
		container.Add(jsonutils.NewStringArray(strings.Split(o.Args, " ")), "args")
	}
	resourcesReq := jsonutils.NewDict()
	if o.Cpu > 0 {
		resourcesReq.Add(jsonutils.NewString(fmt.Sprintf("%dm", int64(o.Cpu*1000))), "cpu")
	}
	if o.Mem > 0 {
		resourcesReq.Add(jsonutils.NewString(fmt.Sprintf("%dMi", o.Mem)), "memory")
	}
	if len(o.Env) != 0 {
		envs := jsonutils.NewArray()
		for _, e := range o.Env {
			parts := strings.Split(e, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("Bad env value: %v", e)
			}
			envObj := jsonutils.NewDict()
			envObj.Add(jsonutils.NewString(parts[0]), "name")
			envObj.Add(jsonutils.NewString(parts[1]), "value")
			envs.Add(envObj)
		}
		container.Add(envs, "env")
	}
	if o.RunAsPrivileged {
		container.Add(jsonutils.JSONTrue, "securityContext", "privileged")
	}
	vols := jsonutils.NewArray()
	volMounts := jsonutils.NewArray()
	if len(o.Pvc) != 0 {
		for _, pvc := range o.Pvc {
			vol, volMount, err := parsePvc(pvc)
			if err != nil {
				return nil, err
			}
			vols.Add(vol)
			volMounts.Add(volMount)
		}
	}
	container.Add(resourcesReq, "resources", "requests")
	if volMounts.Length() > 0 {
		container.Add(volMounts, "volumeMounts")
	}

	containers.Add(container)
	if o.RestartPolicy != "" {
		params.Add(jsonutils.NewString(o.RestartPolicy), "restartPolicy")
	}
	if len(o.RegistrySecret) != 0 {
		rs := jsonutils.NewArray()
		for _, s := range o.RegistrySecret {
			obj := jsonutils.NewDict()
			obj.Add(jsonutils.NewString(s), "name")
			rs.Add(obj)
		}

		params.Add(rs, "imagePullSecrets")
	}
	params.Add(containers, "containers")
	if vols.Length() > 0 {
		params.Add(vols, "volumes")
	}
	return params, nil
}

func parsePvc(pvcDesc string) (jsonutils.JSONObject, jsonutils.JSONObject, error) {
	parts := strings.Split(pvcDesc, ":")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("Invalid PVC desc string: %s", pvcDesc)
	}
	pvcName := parts[0]
	pvcMntPath := parts[1]

	pvcVol := jsonutils.NewDict()
	pvcVol.Add(jsonutils.NewString(pvcName), "claimName")
	vol := jsonutils.NewDict()
	vol.Add(jsonutils.NewString(pvcName), "name")
	vol.Add(pvcVol, "persistentVolumeClaim")

	volMnt := jsonutils.NewDict()
	volMnt.Add(jsonutils.NewString(pvcName), "name")
	volMnt.Add(jsonutils.NewString(pvcMntPath), "mountPath")

	return vol, volMnt, nil
}

type IOption interface {
	Params() (*jsonutils.JSONDict, error)
}

func attachData(o IOption, data *jsonutils.JSONDict, keys ...string) error {
	ret, err := o.Params()
	if err != nil {
		return err
	}
	if ret == nil {
		return nil
	}
	data.Add(ret, keys...)
	return nil
}

func (o K8sPodTemplateOptions) Attach(data *jsonutils.JSONDict) error {
	return attachData(o, data, "template", "spec")
}

type K8sLabelOptions struct {
	Label []string `help:"Labels to apply to the pod(s), e.g. 'env=prod'"`
}

func (o K8sLabelOptions) Params() (*jsonutils.JSONDict, error) {
	labels := map[string]string{}
	for _, label := range o.Label {
		k, v, err := parseLabel(label)
		if err != nil {
			return nil, err
		}
		labels[k] = v
	}
	params := jsonutils.Marshal(labels).(*jsonutils.JSONDict)
	return params, nil
}

func parseLabel(str string) (string, string, error) {
	parts := strings.Split(str, "=")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Invalid label string: %s", str)
	}
	return parts[0], parts[1], nil
}

func (o K8sLabelOptions) Attach(data *jsonutils.JSONDict) error {
	if len(o.Label) == 0 {
		return nil
	}
	return attachData(o, data, "labels")
}

type K8sPVCTemplateOptions struct {
	PvcTemplate []string `help:"PVC volume desc, format is <pvc_name>:<size>:<mount_point>"`
}

func (o K8sPVCTemplateOptions) Parse() ([]*pvcTemplate, error) {
	if len(o.PvcTemplate) == 0 {
		return nil, nil
	}
	pvcs := []*pvcTemplate{}
	for _, pvc := range o.PvcTemplate {
		template, err := parsePVCTemplate(pvc)
		if err != nil {
			return nil, err
		}
		pvcs = append(pvcs, template)
	}
	return pvcs, nil
}

// PVCTemplateOptions Attach must invoke before podTemplate Attach
func (o K8sPVCTemplateOptions) Attach(
	data *jsonutils.JSONDict,
	pvcs []*pvcTemplate,
	podTemplate *K8sPodTemplateOptions,
) {
	pvcsObj := jsonutils.NewArray()
	for _, p := range pvcs {
		pvcsObj.Add(p.pvc)
		podTemplate.Pvc = append(podTemplate.Pvc, p.volMount)
	}
	data.Add(pvcsObj, "volumeClaimTemplates")
}

type pvcTemplate struct {
	pvc      *jsonutils.JSONDict
	volMount string
}

func parsePVCTemplate(pvcDesc string) (*pvcTemplate, error) {
	parts := strings.Split(pvcDesc, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Invalid PVC desc string: %s", pvcDesc)
	}
	pvcName := parts[0]
	pvcSize := parts[1]
	pvcMntPath := parts[2]
	spec := jsonutils.NewDict()
	spec.Add(jsonutils.NewString(pvcSize), "resources", "requests", "storage")
	obj := jsonutils.NewDict()
	obj.Add(spec, "spec")
	obj.Add(jsonutils.NewString(pvcName), "metadata", "name")
	return &pvcTemplate{
		pvc:      obj,
		volMount: fmt.Sprintf("%s:%s", pvcName, pvcMntPath),
	}, nil
}
