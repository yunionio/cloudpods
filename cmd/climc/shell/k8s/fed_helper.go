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

	"github.com/ghodss/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type fedResourceCmd struct {
	*K8sResourceCmd
}

func newFedResourceCmd(manager modulebase.IBaseManager) *fedResourceCmd {
	cmd := NewK8sResourceCmd(manager)
	return &fedResourceCmd{
		K8sResourceCmd: cmd,
	}
}

func (c *fedResourceCmd) List(args shell.IListOpt) *fedResourceCmd {
	c.K8sResourceCmd.List(args)
	return c
}

func (c *fedResourceCmd) Show(args shell.IShowOpt) *fedResourceCmd {
	c.K8sResourceCmd.Show(args)
	return c
}

func (c *fedResourceCmd) Create(args shell.ICreateOpt) *fedResourceCmd {
	c.K8sResourceCmd.Create(args)
	return c
}

func (c *fedResourceCmd) Delete(args shell.IDeleteOpt) *fedResourceCmd {
	c.K8sResourceCmd.Delete(args)
	return c
}

func (c *fedResourceCmd) AttachCluster(args shell.IPerformOpt) *fedResourceCmd {
	c.K8sResourceCmd.Perform("attach-cluster", args)
	return c
}

func (c *fedResourceCmd) DetachCluster(args shell.IPerformOpt) *fedResourceCmd {
	c.K8sResourceCmd.Perform("detach-cluster", args)
	return c
}

func (c *fedResourceCmd) SyncCluster(args shell.IPerformOpt) *fedResourceCmd {
	c.K8sResourceCmd.Perform("sync-cluster", args)
	return c
}

func (c *fedResourceCmd) Sync(args shell.IPerformOpt) *fedResourceCmd {
	c.K8sResourceCmd.Perform("sync", args)
	return c
}

type iFedResUpdateOpt interface {
	shell.IShowOpt
	GetUpdateFields() []string
}

func (c *fedResourceCmd) Update(args iFedResUpdateOpt) *fedResourceCmd {
	man := c.manager
	fields := args.GetUpdateFields()
	callback := func(s *mcclient.ClientSession, args iFedResUpdateOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).Get(s, args.GetId(), params)
		if err != nil {
			return err
		}
		updateData := jsonutils.NewDict()
		for _, field := range fields {
			up, err := ret.Get(field)
			if err != nil {
				return errors.Wrapf(err, "get update field %s", field)
			}
			updateData.Add(up, field)
		}
		content, err := FileTempEdit(args.GetId(), "yaml", updateData.YAMLString())
		if err != nil {
			return errors.Wrap(err, "edit tempfile")
		}
		jsonBytes, err := yaml.YAMLToJSON([]byte(content))
		if err != nil {
			return errors.Wrap(err, "yaml to json")
		}
		updateBody, err := jsonutils.Parse(jsonBytes)
		if err != nil {
			return errors.Wrap(err, "parse json bytes")
		}
		ret, err = man.(modulebase.Manager).Update(s, args.GetId(), updateBody)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	c.RunWithDesc("update", fmt.Sprintf("Update %s of a %s", fields, man.GetKeyword()), args, callback)
	return c
}
