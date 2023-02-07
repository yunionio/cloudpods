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

package options

import (
	"time"

	"yunion.io/x/jsonutils"
)

type WebConsoleOptions struct {
	WebconsoleUrl string `help:"Frontend webconsole url" short-token:"w" default:"$WEBCONSOLE_URL"`
}

type PodBaseOptions struct {
	WebConsoleOptions
	NAME      string `help:"Name of k8s pod to connect"`
	Namespace string `help:"Namespace of this pod"`
	Container string `help:"Container in this pod"`
	Cluster   string `default:"$K8S_CLUSTER|default" help:"Kubernetes cluster name"`
}

func (opt *PodBaseOptions) Params() (*jsonutils.JSONDict, error) {
	return StructToParams(opt)
}

type PodShellOptions struct {
	PodBaseOptions
}

type PodLogOptoins struct {
	PodBaseOptions
	Since string `help:"Only return logs newer than a relative duration like 5s, 2m or 3h"`
}

func (opt *PodLogOptoins) Params() (*jsonutils.JSONDict, error) {
	params, err := opt.PodBaseOptions.Params()
	if err != nil {
		return nil, err
	}
	if opt.Since != "" {
		_, err = time.ParseDuration(opt.Since)
		if err != nil {
			return nil, err
		}
		params.Add(jsonutils.NewString(opt.Since), "since")
	}
	return params, nil
}

type WebConsoleBaremetalOptions struct {
	WebConsoleOptions
	ID string `help:"Baremetal host id or name"`
}

func (opt *WebConsoleBaremetalOptions) Params() (*jsonutils.JSONDict, error) {
	return StructToParams(opt)
}

type WebConsoleSshOptions struct {
	WebConsoleOptions
	IP       string `help:"IP to connect" json:"-"`
	Port     int    `help:"Remote server port"`
	Username string `help:"Remote server username"`
	Password string `help:"Remote server password"`
}

func (opt *WebConsoleSshOptions) Params() (*jsonutils.JSONDict, error) {
	data, err := StructToParams(opt)
	if err != nil {
		return nil, err
	}
	params := jsonutils.NewDict()
	params.Set("webconsole", data)
	return params, nil
}

type WebConsoleServerOptions struct {
	WebConsoleOptions
	ID string `help:"Server id or name"`
}
