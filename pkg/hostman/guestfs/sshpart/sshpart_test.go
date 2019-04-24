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

package sshpart

// TODO: rewrite this test
/*
import (
	//"syscall"
	"testing"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

var defaultSSHClient *ssh.Client

func init() {
	var err error
	defaultSSHClient, err = ssh.NewClient("192.168.0.254", 22, "root", "oYA79tAcs4A1X4uT", "")
	if err != nil {
		log.Fatalf("Connect to ssh client error: %v", err)
	}
}

func TestNewSSHPartition(t *testing.T) {
	dev := NewSSHPartition(defaultSSHClient, "/dev/sda2")
	//err := dev.Mount()
	//if err != nil {
	//t.Errorf("Mount error: %v", err)
	//return
	//}
	//defer func() { dev.Unmount() }()
	log.Infof("%v", dev.Exists("/etc/", false))
	log.Infof("%v", dev.ListDir("/", false))
	dev.Mkdir("/tmp/test123", 0777, false)
	dev.FilePutContents("/tmp/test123/content", "test1234\nhhhh", true, false)
	pubkeys := &sshkeys.SSHKeys{
		PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCz0BLJD+xXYd3AP26uFs42mQSoznPew6gC84P9eUEAJHdkT/8WqTJV0z9M8ZU+8UbuR3iTSbblatrZepPkU2KkvE9ZkFftCIGCWCgvRWFfrDdMF1jwGYtKDg1xVxCmxzTgR+NCuE7HIyDsNL/IKbIVH6QMCxwAIdxHrAT4WdVvkDrD5ihSmIMgnmbCSidok8N7l9zECN54EccV3LGaABumtO5Y7Um7HRm+gdc6esg3HTkIXW402w92zaeHaqm4EGek/FB24WhIcwSErMhXnnHPoAATNzWD+3RQZo2po+95FE/oZw7QO7hG9lWmCDYpJNim+Ix35ftYs1j1S4hray3z lzx@lzx-t470p",
	}
	err := fsdriver.DeployAuthorizedKeys(dev, "/home/cloudroot", pubkeys, true)
	if err != nil {
		log.Errorf("Deploy keys error: %v", err)
	}
}*/
