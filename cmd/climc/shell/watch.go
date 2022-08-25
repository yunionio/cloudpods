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

package shell

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type eventHandler struct {
	man informer.IResourceManager
}

func (e eventHandler) keyword() string {
	return e.man.GetKeyword()
}

func (e eventHandler) OnAdd(obj *jsonutils.JSONDict) {
	log.Infof("%s [CREATED]: \n%s", e.keyword(), obj.String())
}

func (e eventHandler) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	log.Infof("%s [UPDATED]: \n[NEW]: %s\n[OLD]: %s", e.keyword(), newObj.String(), oldObj.String())
}

func (e eventHandler) OnDelete(obj *jsonutils.JSONDict) {
	log.Infof("%s [DELETED]: \n%s", e.keyword(), obj.String())
}

func init() {
	type WatchOptions struct {
		Resource []string `help:"Resource manager plural keyword, e.g.'servers, disks, guestdisks'" short-token:"s"`
		All      bool     `help:"Watch all resources"`
	}

	R(&WatchOptions{}, "watch", "Watch resources", func(s *mcclient.ClientSession, opts *WatchOptions) error {
		watchMan, err := informer.NewWatchManagerBySession(s)
		if err != nil {
			return err
		}
		resources := opts.Resource
		if opts.All {
			resSets := sets.NewString()
			mods, _ := modulebase.GetRegisterdModules()
			resSets.Insert(mods...)
			resources = resSets.List()
		}
		if len(resources) == 0 {
			return errors.Errorf("no watch resources specified")
		}
		for _, res := range resources {
			var resMan informer.IResourceManager
			if modMan, _ := modulebase.GetModule(s, res); modMan != nil {
				resMan = modMan
			}
			if resMan == nil {
				if jointModMan, _ := modulebase.GetJointModule(s, res); jointModMan != nil {
					resMan = jointModMan
				}
			}
			if resMan == nil {
				//return errors.Errorf("Not found %q resource manager", res)
				log.Warningf("Not found %q resource manager", res)
				continue
			}
			if err := watchMan.For(resMan).AddEventHandler(context.Background(), eventHandler{resMan}); err != nil {
				return errors.Wrapf(err, "watch resource %s", res)
			}
		}
		select {}
	})
}
