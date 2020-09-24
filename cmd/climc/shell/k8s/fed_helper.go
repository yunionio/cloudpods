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
	"yunion.io/x/onecloud/cmd/climc/shell"
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
