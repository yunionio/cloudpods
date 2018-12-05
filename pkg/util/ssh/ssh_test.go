package ssh

import (
	"testing"

	"yunion.io/x/log"
)

const (
	username = "root"
	host     = "10.168.222.245"
	password = "123@openmag"
)

func TestRun(t *testing.T) {
	client, err := NewClient(host, 22, username, password, "")
	if err != nil {
		t.Error(err)
	}
	out, err := client.Run("ls", "uname -a", "date", "hostname")
	if err != nil {
		t.Error(err)
	}
	log.Infof("output: %#v", out)
}
