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
	"os"
	"os/exec"

	"github.com/ghodss/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/cmd/climc/shell/events"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

type K8sResourceCmd struct {
	*shell.ResourceCmd
	manager modulebase.IBaseManager
}

func NewK8sResourceCmd(manager modulebase.IBaseManager) *K8sResourceCmd {
	cmd := new(K8sResourceCmd)
	cmd.ResourceCmd = shell.NewResourceCmd(manager).SetPrefix("k8s")
	cmd.manager = manager
	return cmd
}

func (cmd *K8sResourceCmd) GetClusterResManager() k8s.IClusterResourceManager {
	return cmd.manager.(k8s.IClusterResourceManager)
}

func (cmd *K8sResourceCmd) ShowEvent() *K8sResourceCmd {
	callback := func(s *mcclient.ClientSession, opt *events.TypeEventListOptions) error {
		args := &events.EventListOptions{BaseEventListOptions: opt.BaseEventListOptions, Id: opt.ID, Type: []string{cmd.manager.GetKeyword()}}
		return events.DoEventList(*k8s.Logs.ResourceManager, s, args)
	}
	cmd.RunWithDesc("event", fmt.Sprintf("Show operation event logs of k8s %s", cmd.manager.GetKeyword()), new(events.TypeEventListOptions), callback)
	return cmd
}

func FileTempEdit(prefix, suffix string, input string) (string, error) {
	tempfile, err := os.CreateTemp("", fmt.Sprintf("k8s-%s*.%s", prefix, suffix))
	if err != nil {
		return "", errors.Wrap(err, "New tempfile")
	}
	defer os.Remove(tempfile.Name())
	if _, err := tempfile.Write([]byte(input)); err != nil {
		return "", errors.Wrap(err, "write tempfile")
	}
	if err := tempfile.Close(); err != nil {
		return "", err
	}
	cmd := exec.Command("vim", tempfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	content, err := os.ReadFile(tempfile.Name())
	if err != nil {
		return "", errors.Wrapf(err, "read file %s", tempfile.Name())
	}
	return string(content), nil
}

func (cmd *K8sResourceCmd) EditRaw(opt shell.IGetOpt) *K8sResourceCmd {
	man := cmd.GetClusterResManager()
	callback := func(s *mcclient.ClientSession, args shell.IGetOpt) error {
		params, err := opt.Params()
		if err != nil {
			return err
		}
		rawData, err := man.GetRaw(s, args.GetId(), params.(*jsonutils.JSONDict))
		if err != nil {
			return err
		}
		yamlBytes := rawData.YAMLString()
		tempfile, err := os.CreateTemp("", fmt.Sprintf("k8s-%s*.yaml", args.GetId()))
		if err != nil {
			return err
		}
		defer os.Remove(tempfile.Name())
		if _, err := tempfile.Write([]byte(yamlBytes)); err != nil {
			return err
		}
		if err := tempfile.Close(); err != nil {
			return err
		}

		cmd := exec.Command("vim", tempfile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return err
		}
		content, err := os.ReadFile(tempfile.Name())
		if err != nil {
			return err
		}
		jsonBytes, err := yaml.YAMLToJSON(content)
		if err != nil {
			return err
		}
		body, err := jsonutils.Parse(jsonBytes)
		if err != nil {
			return err
		}
		params, err = args.Params()
		if err != nil {
			return err
		}
		if _, err := man.UpdateRaw(s, args.GetId(), params.(*jsonutils.JSONDict), body.(*jsonutils.JSONDict)); err != nil {
			return err
		}
		return nil
	}
	cmd.RunWithDesc("edit-raw", fmt.Sprintf("Edit and update k8s %s by raw data", man.GetKeyword()), opt, callback)
	return cmd
}

func (cmd *K8sResourceCmd) ShowRaw(args shell.IGetOpt) *K8sResourceCmd {
	man := cmd.GetClusterResManager()
	callback := func(s *mcclient.ClientSession, args shell.IGetOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.GetRaw(s, args.GetId(), params.(*jsonutils.JSONDict))
		if err != nil {
			return err
		}
		printObjectYAML(ret)
		return nil
	}
	cmd.RunWithDesc("show-raw", fmt.Sprintf("Show k8s %s raw data", man.GetKeyword()), args, callback)
	return cmd
}

func NewK8sJointCmd(manager modulebase.JointManager) *shell.JointCmd {
	cmd := shell.NewJointCmd(manager)
	cmd.SetPrefix("k8s")
	return cmd
}
