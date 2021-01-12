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

package ssh

import (
	"context"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/pkg/errors"
)

type ClientConfig struct {
	User string
	Host string
	Port int
	Key  string
}

func (cc *ClientConfig) NewClient(ctx context.Context) (*ssh.Client, error) {
	signer, err := ssh.ParsePrivateKey([]byte(cc.Key))
	if err != nil {
		return nil, errors.Wrap(err, "parse ssh key")
	}
	sshcc := &ssh.ClientConfig{
		User: cc.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := net.JoinHostPort(cc.Host, fmt.Sprintf("%d", cc.Port))
	d := &net.Dialer{}
	netconn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "net dial")
	}

	sshconn, chans, reqs, err := ssh.NewClientConn(netconn, addr, sshcc)
	if err != nil {
		netconn.Close()
		return nil, errors.Wrap(err, "ssh new client conn")
	}

	sshc := ssh.NewClient(sshconn, chans, reqs)
	return sshc, nil
}
