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

package compute

import (
	"fmt"
	"os"
	"path"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	type SshkeypairQueryOptions struct {
		Project string `help:"get keypair for specific project"`
		Admin   bool   `help:"get admin keypair, sysadmin ONLY option"`
	}

	getSshKeypair := func(s *mcclient.ClientSession, args *SshkeypairQueryOptions) (string, string, error) {
		query := jsonutils.NewDict()
		if args.Admin {
			query.Add(jsonutils.JSONTrue, "admin")
		}
		var keys jsonutils.JSONObject
		if len(args.Project) == 0 {
			listResult, err := modules.Sshkeypairs.List(s, query)
			if err != nil {
				return "", "", err
			}
			keys = listResult.Data[0]
		} else {
			result, err := modules.Sshkeypairs.GetById(s, args.Project, query)
			if err != nil {
				return "", "", err
			}
			keys = result
		}
		privKey, _ := keys.GetString("private_key")
		pubKey, _ := keys.GetString("public_key")
		return privKey, pubKey, nil
	}

	R(&SshkeypairQueryOptions{}, "sshkeypair-show", "Get ssh keypairs", func(s *mcclient.ClientSession, args *SshkeypairQueryOptions) error {
		privKey, pubKey, err := getSshKeypair(s, args)
		if err != nil {
			return err
		}

		fmt.Print(privKey)
		fmt.Print(pubKey)

		return nil
	})

	type SshkeypairInjectOptions struct {
		SshkeypairQueryOptions
		TargetDir string `help:"Target directory to save cloud ssh keypair"`
	}
	R(&SshkeypairInjectOptions{}, "sshkeypair-inject", "Inject ssh keypairs to local path", func(s *mcclient.ClientSession, args *SshkeypairInjectOptions) error {
		_, pubKey, err := getSshKeypair(s, &args.SshkeypairQueryOptions)
		if err != nil {
			return err
		}
		targetDir := args.TargetDir
		if targetDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return errors.Wrap(err, "get current user's home dir")
			}
			targetDir = homeDir
		}
		sshDir := path.Join(targetDir, ".ssh")
		// MkdirAll anyways
		os.MkdirAll(sshDir, 0700)
		authFile := path.Join(sshDir, "authorized_keys")
		var oldKeys string

		if procutils.NewCommand("test", "-f", authFile).Run() == nil {
			output, err := procutils.NewCommand("cat", authFile).Output()
			if err != nil {
				return errors.Wrapf(err, "cat: %s", output)
			}
			oldKeys = string(output)
		}
		var MergeAuthorizedKeys = func(oldKeys string, pubKey string) string {
			const sshKeySignature = "@yunioncloudpods"
			var allkeys = make(map[string]string)
			if len(oldKeys) > 0 {
				for _, line := range strings.Split(oldKeys, "\n") {
					line = strings.TrimSpace(line)
					dat := strings.Split(line, " ")
					if len(dat) > 1 {
						if len(dat) > 2 && dat[2] == sshKeySignature {
							// skip ssh keys with signature
							continue
						}
						if _, ok := allkeys[dat[1]]; !ok {
							allkeys[dat[1]] = line
						}
					}
				}
			}
			candiateKeys := []string{pubKey}
			for _, k := range candiateKeys {
				if len(k) > 0 {
					k = strings.TrimSpace(k)
					dat := strings.Split(k, " ")
					if len(dat) > 1 {
						if _, ok := allkeys[dat[1]]; !ok {
							allkeys[dat[1]] = strings.Join([]string{dat[0], dat[1], sshKeySignature}, " ")
						}
					}
				}
			}
			var keys = make([]string, 0)
			for _, val := range allkeys {
				keys = append(keys, val)
			}
			return strings.Join(keys, "\n") + "\n"
		}

		newKeys := MergeAuthorizedKeys(oldKeys, pubKey)
		if output, err := procutils.NewCommand(
			"sh", "-c", fmt.Sprintf("echo '%s' > %s", newKeys, authFile)).Output(); err != nil {
			return errors.Wrapf(err, "write public keys: %s", output)
		}
		if output, err := procutils.NewCommand(
			"chmod", "0644", authFile).Output(); err != nil {
			return errors.Wrapf(err, "chmod failed %s", output)
		}
		return nil
	})
}
