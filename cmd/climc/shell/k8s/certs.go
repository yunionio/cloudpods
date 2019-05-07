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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initKubeCerts() {
	cmdN := func(action string) string {
		return fmt.Sprintf("kubecert-%s", action)
	}
	R(&o.CertListOptions{}, cmdN("list"), "List k8s cluster certificates key pairs", func(s *mcclient.ClientSession, args *o.CertListOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := k8s.KubeCerts.List(s, params)
		if err != nil {
			return err
		}
		printList(ret, k8s.KubeCerts.GetColumns(s))
		return nil
	})
}
