package ansible

import (
	"context"
	"os/exec"
	"reflect"
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
		{
			Name: "copy",
			Args: []string{
				"src=afile",
				"dest=/tmp/afile",
			},
		},
		{
			Name: "copy",
			Args: []string{
				"src=adir/afile",
				"dest=/tmp/adirfile",
			},
		},
	}
	pb.Files = map[string][]byte{
		"afile":      []byte("afilecontent"),
		"adir/afile": []byte("afilecontent under adir"),
	}

	t.Run("copy", func(t *testing.T) {
		pb2 := pb.Copy()
		if !reflect.DeepEqual(pb2, pb) {
			t.Errorf("copy and the original should be equal")
		}
	})
	t.Run("run", func(t *testing.T) {
		err := pb.Run(context.TODO())
		t.Logf("%s", pb.Output())
		if err != nil {
			t.Fatalf("not expecting err: %v", err)
		}
	})
}
