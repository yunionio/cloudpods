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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initRepo() {
	cmdN := func(suffix string) string {
		return resourceCmdN("repo", suffix)
	}
	R(&o.RepoListOptions{}, cmdN("list"), "List k8s global helm repos", func(s *mcclient.ClientSession, args *o.RepoListOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		result, err := k8s.Repos.List(s, params)
		if err != nil {
			return err
		}
		printList(result, k8s.Repos.GetColumns(s))
		return nil
	})

	R(&o.RepoGetOptions{}, cmdN("show"), "Show details of a repo", func(s *mcclient.ClientSession, args *o.RepoGetOptions) error {
		repo, err := k8s.Repos.Get(s, args.NAME, nil)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	R(&o.RepoCreateOptions{}, cmdN("create"), "Add repository", func(s *mcclient.ClientSession, args *o.RepoCreateOptions) error {
		repo, err := k8s.Repos.Create(s, args.Params())
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	R(&o.RepoUpdateOptions{}, cmdN("update"), "Update helm repository", func(s *mcclient.ClientSession, args *o.RepoUpdateOptions) error {
		repo, err := k8s.Repos.Update(s, args.NAME, args.Params())
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	R(&o.RepoGetOptions{}, cmdN("delete"), "Delete a repository", func(s *mcclient.ClientSession, args *o.RepoGetOptions) error {
		repo, err := k8s.Repos.Delete(s, args.NAME, nil)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	R(&o.RepoGetOptions{}, cmdN("sync"), "Sync a repository", func(s *mcclient.ClientSession, args *o.RepoGetOptions) error {
		repo, err := k8s.Repos.PerformAction(s, args.NAME, "sync", nil)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	R(&o.RepoGetOptions{}, cmdN("public"), "Make repository public", func(s *mcclient.ClientSession, args *o.RepoGetOptions) error {
		repo, err := k8s.Repos.PerformAction(s, args.NAME, "public", nil)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})

	R(&o.RepoGetOptions{}, cmdN("private"), "Make repository private", func(s *mcclient.ClientSession, args *o.RepoGetOptions) error {
		repo, err := k8s.Repos.PerformAction(s, args.NAME, "private", nil)
		if err != nil {
			return err
		}
		printObject(repo)
		return nil
	})
}
