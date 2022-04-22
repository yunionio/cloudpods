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

package command

import (
	"reflect"
	"strings"
	"testing"

	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

func init() {
	o.Options.KubectlPath = "/usr/bin/kubectl"
}

func TestKubectlExec_Command(t *testing.T) {
	type fields struct {
		Kubectl *Kubectl
	}
	type args struct {
		cmd  string
		args []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "bash command",
			fields: fields{
				Kubectl: NewKubectlCommand(nil, "/tmp/kubeconfig", "system"),
			},
			args: args{
				cmd:  "bash",
				args: []string{"-il"},
			},
			want: "/usr/bin/kubectl --namespace system exec -i -t Pod1 -c Container1 -- bash -il",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fields.Kubectl.Exec().
				Stdin().TTY().
				Pod("Pod1").
				Container("Container1")
			cmd := c.Command(tt.args.cmd, tt.args.args...)
			name := cmd.name
			args := cmd.args
			got := []string{name}
			got = append(got, args...)
			if !reflect.DeepEqual(strings.Join(got, " "), tt.want) {
				t.Errorf("KubectlExec.Command() = %v, want %v", got, tt.want)
			}
		})
	}
}
