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
