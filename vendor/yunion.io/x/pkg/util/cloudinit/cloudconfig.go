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
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"
)

/*
 * cloudconfig
 * Reference:  https://cloudinit.readthedocs.io/en/latest/topics/examples.html
 *
 */

type TSudoPolicy string
type TSshPwauth string

const (
	CLOUD_CONFIG_HEADER      = "#cloud-config\n"
	CLOUD_SHELL_HEADER       = "#!/usr/bin/env bash\n"
	CLOUD_POWER_SHELL_HEADER = "#ps1\n"

	USER_SUDO_NOPASSWD = TSudoPolicy("sudo_nopasswd")
	USER_SUDO          = TSudoPolicy("sudo")
	USER_SUDO_DENY     = TSudoPolicy("sudo_deny")
	USER_SUDO_NONE     = TSudoPolicy("")

	SSH_PASSWORD_AUTH_ON        = TSshPwauth("true")
	SSH_PASSWORD_AUTH_OFF       = TSshPwauth("false")
	SSH_PASSWORD_AUTH_UNCHANGED = TSshPwauth("unchanged")
)

type SWriteFile struct {
	Path        string
	Permissions string
	Owner       string
	Encoding    string
	Content     string
}

type SUser struct {
	Name              string
	PlainTextPasswd   string
	HashedPasswd      string
	LockPasswd        bool
	SshAuthorizedKeys []string
	Sudo              string
}

type SPhoneHome struct {
	Url string
}

type SCloudConfig struct {
	Users       []SUser
	WriteFiles  []SWriteFile
	Runcmd      []string
	Bootcmd     []string
	Packages    []string
	PhoneHome   *SPhoneHome
	DisableRoot int
	SshPwauth   TSshPwauth
}

func NewWriteFile(path string, content string, perm string, owner string, isBase64 bool) SWriteFile {
	f := SWriteFile{}

	f.Path = path
	f.Permissions = perm
	f.Owner = owner
	if isBase64 {
		f.Encoding = "b64"
		f.Content = base64.StdEncoding.EncodeToString([]byte(content))
	} else {
		f.Content = content
	}

	return f
}

func setFilePermission(path, permission, owner string) []string {
	cmds := []string{}
	if len(permission) > 0 {
		cmds = append(cmds, fmt.Sprintf("chmod %s %s", permission, path))
	}
	if len(owner) > 0 {
		cmds = append(cmds, fmt.Sprintf("chown %s:%s %s", owner, owner, path))
	}
	return cmds
}

func mkPutFileCmd(path string, content string, permission string, owner string) []string {
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("mkdir -p $(dirname %s)", path))
	cmds = append(cmds, fmt.Sprintf("cat > %s <<_END\n%s\n_END", path, content))
	return append(cmds, setFilePermission(path, permission, owner)...)
}

func mkAppendFileCmd(path string, content string, permission string, owner string) []string {
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("mkdir -p $(dirname %s)", path))
	cmds = append(cmds, fmt.Sprintf("cat >> %s <<_END\n%s\n_END", path, content))
	return append(cmds, setFilePermission(path, permission, owner)...)
}

func (wf *SWriteFile) ShellScripts() []string {
	content := wf.Content
	if wf.Encoding == "b64" {
		_content, _ := base64.StdEncoding.DecodeString(wf.Content)
		content = string(_content)
	}

	return mkPutFileCmd(wf.Path, content, wf.Permissions, wf.Owner)
}

func NewUser(name string) SUser {
	u := SUser{Name: name}
	return u
}

func (u *SUser) SudoPolicy(policy TSudoPolicy) *SUser {
	switch policy {
	case USER_SUDO_NOPASSWD:
		u.Sudo = "ALL=(ALL) NOPASSWD:ALL"
	case USER_SUDO:
		u.Sudo = "ALL=(ALL) ALL"
	case USER_SUDO_DENY:
		u.Sudo = "False"
	default:
		u.Sudo = ""
	}
	return u
}

func (u *SUser) SshKey(key string) *SUser {
	if u.SshAuthorizedKeys == nil {
		u.SshAuthorizedKeys = make([]string, 0)
	}
	u.SshAuthorizedKeys = append(u.SshAuthorizedKeys, key)
	return u
}

func (u *SUser) Password(passwd string) *SUser {
	if len(passwd) > 0 {
		hash, err := seclib.GeneratePassword(passwd)
		if err != nil {
			log.Errorf("GeneratePassword error %s", err)
		} else {
			u.PlainTextPasswd = passwd
			u.HashedPasswd = hash
		}
		u.LockPasswd = false
	}
	return u
}

func (u *SUser) PowerShellScripts() []string {
	shells := []string{}
	shells = append(shells, fmt.Sprintf(`New-LocalUser -Name "%s" -Description "A New Local Account Created By PowerShell" -NoPassword`, u.Name))
	shells = append(shells, fmt.Sprintf(`Add-LocalGroupMember -Group "Administrators" -Member "%s"`, u.Name))
	if len(u.PlainTextPasswd) > 0 {
		shells = append(shells, fmt.Sprintf(`net user "%s" "%s"`, u.Name, u.PlainTextPasswd))
	}
	// enable需要再设置密码之后，否则会出现Enable-LocalUser : Unable to update the password. The value provided for the new password does not meet the length, complexity, or history requirements of the domain
	shells = append(shells, fmt.Sprintf(`Enable-LocalUser "%s"`, u.Name))
	return shells
}

func (u *SUser) ShellScripts() []string {
	shells := []string{}

	shells = append(shells, fmt.Sprintf("useradd -m %s || true", u.Name))
	if len(u.HashedPasswd) > 0 {
		shells = append(shells, fmt.Sprintf("usermod -p '%s' %s", u.HashedPasswd, u.Name))
	}

	home := "/" + u.Name
	if home != "/root" {
		home = "/home" + home
	}

	keyPath := fmt.Sprintf("%s/.ssh/authorized_keys", home)
	shells = append(shells, mkAppendFileCmd(keyPath, strings.Join(u.SshAuthorizedKeys, "\n"), "600", u.Name)...)
	shells = append(shells, fmt.Sprintf("chown -R %s:%s %s/.ssh", u.Name, u.Name, home))

	if !utils.IsInStringArray(u.Sudo, []string{"", "False"}) {
		shells = append(shells, mkPutFileCmd("/etc/sudoers.d/"+u.Name, fmt.Sprintf("%s	%s", u.Name, u.Sudo), "", "")...)
	}

	return shells
}

func (conf *SCloudConfig) UserData() string {
	var buf bytes.Buffer
	jsonConf := jsonutils.Marshal(conf).(*jsonutils.JSONDict)
	if jsonConf.Contains("users") {
		userArray := jsonutils.NewArray(jsonutils.NewString("default"))
		users, _ := jsonConf.GetArray("users")
		if users != nil {
			userArray.Add(users...)
			jsonConf.Set("users", userArray)
		}
	}
	buf.WriteString(CLOUD_CONFIG_HEADER)
	buf.WriteString(jsonConf.YAMLString())
	return buf.String()
}

func (conf *SCloudConfig) UserDataScript() string {
	shells := []string{}
	for _, u := range conf.Users {
		shells = append(shells, u.ShellScripts()...)
	}
	shells = append(shells, conf.Runcmd...)

	// 允许密码及root登录(谷歌云镜像默认会禁止root及密码登录)
	shells = append(shells, `sed -i "s/.*PermitRootLogin.*/PermitRootLogin yes/g" /etc/ssh/sshd_config`)
	shells = append(shells, `sed -i 's/.*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config`)
	shells = append(shells, `systemctl restart sshd`)

	for _, pkg := range conf.Packages {
		shells = append(shells, "which yum &>/dev/null && yum install -y "+pkg)
		shells = append(shells, "which apt-get &>/dev/null && apt-get install -y "+pkg)
	}
	for _, wf := range conf.WriteFiles {
		shells = append(shells, wf.ShellScripts()...)
	}
	return CLOUD_SHELL_HEADER + strings.Join(shells, "\n")
}

func (conf *SCloudConfig) UserDataPowerShell() string {
	shells := []string{}
	for _, u := range conf.Users {
		shells = append(shells, u.PowerShellScripts()...)
	}
	shells = append(shells, conf.Runcmd...)

	return CLOUD_POWER_SHELL_HEADER + strings.Join(shells, "\n")
}

func (conf *SCloudConfig) UserDataEc2() string {
	shells := []string{}
	for _, u := range conf.Users {
		shells = append(shells, u.PowerShellScripts()...)
	}
	shells = append(shells, conf.Runcmd...)
	return "<powershell>\n" + strings.Join(shells, "\n") + "\n</powershell>"
}

func (conf *SCloudConfig) UserDataBase64() string {
	data := conf.UserData()
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func (conf *SCloudConfig) UserDataScriptBase64() string {
	data := conf.UserDataScript()
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func ParseUserDataBase64(b64data string) (*SCloudConfig, error) {
	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return nil, err
	}
	return ParseUserData(string(data))
}

func ParseUserData(data string) (*SCloudConfig, error) {
	if !strings.HasPrefix(data, CLOUD_CONFIG_HEADER) {
		msg := "invalid userdata, not starting with #cloud-config"
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}
	jsonConf, err := jsonutils.ParseYAML(data)
	if err != nil {
		log.Errorf("parse userdata yaml error %s", err)
		return nil, err
	}
	jsonDict := jsonConf.(*jsonutils.JSONDict)
	if jsonDict.Contains("users") {
		userArray := jsonutils.NewArray()
		users, _ := jsonConf.GetArray("users")
		if users != nil {
			for i := 0; i < len(users); i++ {
				if users[i].String() != `"default"` {
					userArray.Add(users[i])
				}
			}
			jsonDict.Set("users", userArray)
		}
	}
	config := SCloudConfig{}
	err = jsonDict.Unmarshal(&config)
	if err != nil {
		log.Errorf("unable to unmarchal userdata %s", err)
		return nil, err
	}
	return &config, nil
}

func (conf *SCloudConfig) MergeUser(u SUser) {
	for i := 0; i < len(conf.Users); i += 1 {
		if u.Name == conf.Users[i].Name {
			// replace conf user password with input
			if len(u.PlainTextPasswd) > 0 {
				conf.Users[i].PlainTextPasswd = u.PlainTextPasswd
				conf.Users[i].HashedPasswd = u.HashedPasswd
				conf.Users[i].LockPasswd = u.LockPasswd
			}

			// find user, merge keys
			for j := 0; j < len(u.SshAuthorizedKeys); j += 1 {
				if !utils.IsInStringArray(u.SshAuthorizedKeys[j], conf.Users[i].SshAuthorizedKeys) {
					conf.Users[i].SshAuthorizedKeys = append(conf.Users[i].SshAuthorizedKeys, u.SshAuthorizedKeys[j])
				}
			}
			return
		}
	}
	// no such user
	conf.Users = append(conf.Users, u)
}

func (conf *SCloudConfig) RemoveUser(u SUser) {
	for i := range conf.Users {
		if u.Name == conf.Users[i].Name {
			if len(conf.Users) == i {
				conf.Users = conf.Users[0:i]
			} else {
				conf.Users = append(conf.Users[0:i], conf.Users[i+1:]...)
			}

			return
		}
	}
}

func (conf *SCloudConfig) MergeWriteFile(f SWriteFile, replace bool) {
	for i := 0; i < len(conf.WriteFiles); i += 1 {
		if conf.WriteFiles[i].Path == f.Path {
			// find file
			if replace {
				conf.WriteFiles[i].Content = f.Content
				conf.WriteFiles[i].Encoding = f.Encoding
				conf.WriteFiles[i].Owner = f.Owner
				conf.WriteFiles[i].Permissions = f.Permissions
			}
			return
		}
	}
	// no such file
	conf.WriteFiles = append(conf.WriteFiles, f)
}

func (conf *SCloudConfig) MergeRuncmd(cmd string) {
	if !utils.IsInStringArray(cmd, conf.Runcmd) {
		conf.Runcmd = append(conf.Runcmd, cmd)
	}
}

func (conf *SCloudConfig) MergeBootcmd(cmd string) {
	if !utils.IsInStringArray(cmd, conf.Bootcmd) {
		conf.Bootcmd = append(conf.Bootcmd, cmd)
	}
}

func (conf *SCloudConfig) MergePackage(pkg string) {
	if !utils.IsInStringArray(pkg, conf.Packages) {
		conf.Packages = append(conf.Packages, pkg)
	}
}

func (conf *SCloudConfig) Merge(conf2 *SCloudConfig) {
	for _, u := range conf2.Users {
		conf.MergeUser(u)
	}
	for _, f := range conf2.WriteFiles {
		conf.MergeWriteFile(f, false)
	}
	for _, c := range conf2.Runcmd {
		conf.MergeRuncmd(c)
	}
	for _, c := range conf2.Bootcmd {
		conf.MergeBootcmd(c)
	}
	for _, p := range conf2.Packages {
		conf.MergePackage(p)
	}
}
