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

package helper

import (
	"context"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

func fetchK8sClimcTargetIp() (string, error) {
	ctx := context.Background()
	adminSession := auth.GetAdminSession(ctx, o.Options.Region)

	query := jsonutils.NewDict()
	query.Add(jsonutils.JSONTrue, "system")
	query.Add(jsonutils.NewString("system"), "scope")
	query.Add(jsonutils.NewString("system-default"), "name")
	clusters, err := k8s.KubeClusters.List(adminSession, query)
	if err != nil {
		return "", errors.Wrap(err, "list k8s cluster")
	}
	clusterId, _ := clusters.Data[0].GetString("id")
	if len(clusterId) == 0 {
		return "", httperrors.NewNotFoundError("cluster system-default no id")
	}
	query = jsonutils.NewDict()
	query.Add(jsonutils.NewString(clusterId), "cluster")
	query.Add(jsonutils.NewString("onecloud"), "namespace")
	query.Add(jsonutils.NewString("climc"), "search")
	query.Add(jsonutils.JSONTrue, "details")
	pods, err := k8s.Pods.List(adminSession, query)
	if err != nil {
		return "", errors.Wrap(err, "Pods")
	}
	if len(pods.Data) == 0 {
		return "", httperrors.NewNotFoundError("pod climc not found")
	}
	pod := pods.Data[0]
	podIp, err := pod.GetString("podIP")
	if err != nil {
		return "", errors.Wrap(err, "get podIP")
	}
	return podIp, nil
}

func FetchClimcTargetIp() string {
	podIp, err := fetchK8sClimcTargetIp()
	if err != nil {
		log.Errorf("fetchK8sClimcTargetIp: %v", err)
		return "climc"
	}
	return podIp
}

func GetValidPrivateKey(host string, port int, username string, cliSession *mcclient.ClientSession) (string, error) {
	errs := []error{}
	ctx := context.Background()
	admin := auth.GetAdminSession(ctx, o.Options.Region)
	var privateKeys []string
	projectPrivateKeys, _ := compute.Sshkeypairs.FetchProjectPrivateKeysBySession(ctx, cliSession)
	privateKeys = append(privateKeys, projectPrivateKeys...)
	adminPrivateKeys, _ := compute.Sshkeypairs.FetchAdminPrivateKeysBySession(ctx, admin)
	privateKeys = append(privateKeys, adminPrivateKeys...)

	for _, privKey := range privateKeys {
		signer, err := ssh.ParsePrivateKey([]byte(privKey))
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ParsePrivateKey"))
			continue
		}
		config := &ssh.ClientConfig{
			Timeout:         time.Second,
			User:            username,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
		}
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "dial %s by %s", addr, username))
			continue
		}
		defer client.Close()
		return privKey, nil
	}
	return "", errors.NewAggregate(errs)
}
