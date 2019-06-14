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

package ansible

import (
	"context"
	"os/exec"
	"testing"
)

func skipIfNoAnsible(t *testing.T) {
	_, err := exec.LookPath("ansible")
	if err != nil {
		t.Skipf("looking for ansible: %v", err)
	}
}

func TestPlaybook(t *testing.T) {
	skipIfNoAnsible(t)

	pb := NewPlaybook()
	pb.Inventory = Inventory{
		Hosts: []Host{
			{
				Name: "127.0.0.1",
				Vars: map[string]string{
					"ansible_connection": "local",
				},
			},
		},
	}
	pb.Modules = []Module{
		{
			Name: "ping",
		},
	}
	err := pb.Run(context.TODO())
	if err != nil {
		t.Fatalf("not expecting err: %v", err)
	}
	t.Logf("%s", pb.Output())
}
