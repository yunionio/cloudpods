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

package cloudinit

import (
	"testing"
)

func TestSCloudConfig_UserData(t *testing.T) {
	usr1 := NewUser("root")
	usr1.SshKey("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCa4E8wmIOlmh1G8ZRcU2zpnl2frD2lLKdXpbTeUUZEKYFFlYM8TM5UrKrqrMCd3rFjaYGTKWiQwOiWroXlAXausbbVEI29KY+1Vd26qNyejj+CZO9MCj0naIrqa1V0of3TQY5I2U+ToIkyLqVFWhWVa57v/GUxsV2aNTmUS/qz0OPSCFPbGWWB35rsjwnFwq2jF6E8yJgTGDTYZcsghRi3IWfyfeHbSuWdvn6N8XrPBDmNg7h+GSvO6FJlp6MUw1hscECi13GwqXYgJnLG5RMiFH6s0vhozyHkue1vOTcryPHRQD0Jz/INUSaggH8L1HnYSUavOf4Cw25W9HfzgUBf")

	usr2 := NewUser("yunion")
	usr2.Password("123@yunion").SudoPolicy(USER_SUDO_NOPASSWD)

	file1 := NewWriteFile("/etc/ansible/hosts", "gobuild\ncloudev\n", "", "", true)
	file2 := NewWriteFile("/etc/hosts", "127.0.0.1 localhost\n", "", "", false)
	config := SCloudConfig{
		Users: []SUser{
			usr1,
			usr2,
		},
		WriteFiles: []SWriteFile{
			file1,
			file2,
		},
		Runcmd: []string{
			"mkdir /var/run/httpd",
		},
		PhoneHome: &SPhoneHome{
			Url: "http://www.yunion.io/$INSTANCE_ID",
		},
		DisableRoot: 0,
		SshPwauth:   SSH_PASSWORD_AUTH_ON,
	}
	userData := config.UserData()

	t.Logf("%s", userData)

	config2, err := ParseUserData(userData)
	if err != nil {
		t.Errorf("%s", err)
	} else {
		userData2 := config2.UserData()
		t.Logf("%s", userData2)

		if userData != userData2 {
			t.Errorf("userData not equal to userData2")
		}
	}
	t.Log(config2.UserDataScript())
}
