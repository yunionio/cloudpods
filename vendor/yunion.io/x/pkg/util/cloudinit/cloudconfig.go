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
	"path"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
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

// shellQuote wraps s so that a POSIX shell treats it as one literal word.
// A single quote inside s is closed, escaped and reopened.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// validUserName reports whether name can be used as a user name.
//
// The set is deliberately narrow. A name becomes a directory under /home, a
// file name under /etc/sudoers.d and an argument to useradd, so a slash or a
// ".." would move the files that are written, and a control character would
// split a generated line. Leading "-" is refused so the name cannot be read as
// an option.
func validUserName(name string) bool {
	if len(name) == 0 || name == "." || name == ".." || name[0] == '-' {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '_' || c == '-' || c == '.':
		default:
			return false
		}
	}
	return true
}

// validWritePath reports whether path may be written by a generated script.
//
// Quoting keeps a path from running as a command, but a path that walks up out
// of its directory would still write somewhere else. A path carrying a ".."
// element is refused rather than normalised, so that what is written is what
// was asked for.
func validWritePath(filePath string) bool {
	if len(filePath) == 0 {
		return false
	}
	for _, elem := range strings.Split(filePath, "/") {
		if elem == ".." {
			return false
		}
	}
	return true
}

// escapePowerShell escapes a value for use inside a PowerShell double quoted
// string, where a backtick introduces an escape, $ introduces a variable, and
// a control character written literally would break the line it sits on.
func escapePowerShell(s string) string {
	return strings.NewReplacer(
		"`", "``",
		`"`, "`\"",
		"$", "`$",
		"\r", "`r",
		"\n", "`n",
		"\t", "`t",
		"\x00", "`0",
	).Replace(s)
}

const heredocPrefix = "_YUNION_EOF_"

// heredocTerminator returns a terminator that does not occur in content, so
// that a line of the content cannot end the heredoc early. Quote it at the use
// site so the shell does not expand the content.
//
// The generated value is checked against the content rather than assumed to be
// unique: utils.GenRequestId returns the empty string if the random source is
// unavailable, which would otherwise leave a fixed, guessable terminator.
func heredocTerminator(content string) string {
	return uniqueTerminator(heredocPrefix+utils.GenRequestId(8), content)
}

// uniqueTerminator returns term with underscores appended until it does not
// occur in content. Each pass lengthens it, so this ends once it is longer
// than the content and can no longer occur in it.
func uniqueTerminator(term, content string) string {
	for strings.Contains(content, term) {
		term += "_"
	}
	return term
}

func setFilePermission(path, permission, owner string) []string {
	cmds := []string{}
	if len(permission) > 0 {
		cmds = append(cmds, fmt.Sprintf("chmod %s %s", shellQuote(permission), shellQuote(path)))
	}
	if len(owner) > 0 {
		cmds = append(cmds, fmt.Sprintf("chown %s:%s %s", shellQuote(owner), shellQuote(owner), shellQuote(path)))
	}
	return cmds
}

// mkWriteFileCmd builds the commands that create path with content. The
// redirection and the path are quoted, and the heredoc is quoted with a
// terminator that is not expected to occur in the content.
func mkWriteFileCmd(redirect, filePath, content, permission, owner string) []string {
	terminator := heredocTerminator(content)
	cmds := []string{
		fmt.Sprintf("mkdir -p %s", shellQuote(path.Dir(filePath))),
		fmt.Sprintf("cat %s %s <<'%s'\n%s\n%s", redirect, shellQuote(filePath), terminator, content, terminator),
	}
	return append(cmds, setFilePermission(filePath, permission, owner)...)
}

func mkPutFileCmd(path string, content string, permission string, owner string) []string {
	return mkWriteFileCmd(">", path, content, permission, owner)
}

func mkAppendFileCmd(path string, content string, permission string, owner string) []string {
	return mkWriteFileCmd(">>", path, content, permission, owner)
}

func (wf *SWriteFile) ShellScripts() []string {
	if !validWritePath(wf.Path) {
		log.Errorf("cloudinit: skipping write_file with a path that walks up out of its directory: %q", wf.Path)
		return nil
	}
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
	if !validUserName(u.Name) {
		log.Errorf("cloudinit: skipping scripts for unusable user name %q", u.Name)
		return nil
	}
	// Every line below is parsed by PowerShell first, so the name and the
	// password are escaped the same way. Using the unescaped name for one of
	// the lines would have them name different accounts.
	name := escapePowerShell(u.Name)
	shells := []string{}
	shells = append(shells, fmt.Sprintf(`New-LocalUser -Name "%s" -Description "A New Local Account Created By PowerShell" -NoPassword`, name))
	shells = append(shells, fmt.Sprintf(`Add-LocalGroupMember -Group "Administrators" -Member "%s"`, name))
	if len(u.PlainTextPasswd) > 0 {
		shells = append(shells, fmt.Sprintf(`net user "%s" "%s"`, name, escapePowerShell(u.PlainTextPasswd)))
	}
	// enable需要再设置密码之后，否则会出现Enable-LocalUser : Unable to update the password. The value provided for the new password does not meet the length, complexity, or history requirements of the domain
	shells = append(shells, fmt.Sprintf(`Enable-LocalUser "%s"`, name))
	return shells
}

func (u *SUser) ShellScripts() []string {
	if !validUserName(u.Name) {
		log.Errorf("cloudinit: skipping scripts for unusable user name %q", u.Name)
		return nil
	}
	name := shellQuote(u.Name)
	shells := []string{}

	shells = append(shells, fmt.Sprintf("useradd -m %s || true", name))
	if len(u.HashedPasswd) > 0 {
		shells = append(shells, fmt.Sprintf("usermod -p %s %s", shellQuote(u.HashedPasswd), name))
	}

	home := "/" + u.Name
	if home != "/root" {
		home = "/home" + home
	}

	keyPath := fmt.Sprintf("%s/.ssh/authorized_keys", home)
	shells = append(shells, mkAppendFileCmd(keyPath, strings.Join(u.SshAuthorizedKeys, "\n"), "600", u.Name)...)
	shells = append(shells, fmt.Sprintf("chown -R %s:%s %s", name, name, shellQuote(home+"/.ssh")))

	if !utils.IsInStringArray(u.Sudo, []string{"", "False"}) {
		shells = append(shells, mkPutFileCmd("/etc/sudoers.d/"+u.Name, fmt.Sprintf("%s	%s", u.Name, u.Sudo), "", "")...)
	}

	return shells
}

func (conf *SCloudConfig) UserData() string {
	if conf == nil {
		return ""
	}
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
	if conf == nil {
		return ""
	}
	shells := []string{}
	for _, u := range conf.Users {
		shells = append(shells, u.ShellScripts()...)
	}
	shells = append(shells, conf.Runcmd...)

	if conf.DisableRoot == 0 {
		shells = append(shells, `sed -i "s/.*PermitRootLogin.*/PermitRootLogin yes/g" /etc/ssh/sshd_config`)
		shells = append(shells, `sed -i "s/.*PermitRootLogin.*/PermitRootLogin yes/g" /etc/ssh/sshd_config.d/*.conf`)
	}
	if conf.SshPwauth == SSH_PASSWORD_AUTH_ON {
		shells = append(shells, `sed -i 's/.*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config`)
		shells = append(shells, `sed -i 's/.*PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config.d/*.conf`)
	}
	if conf.DisableRoot == 0 || conf.SshPwauth == SSH_PASSWORD_AUTH_ON {
		// ubuntu24.04 sshd -> ssh
		shells = append(shells, `systemctl restart sshd ssh`)
	}

	for _, pkg := range conf.Packages {
		// A package name is an argument, not part of the command line, so it
		// is quoted rather than pasted in.
		quoted := shellQuote(pkg)
		shells = append(shells, "which yum &>/dev/null && yum install -y "+quoted)
		shells = append(shells, "which apt-get &>/dev/null && apt-get install -y "+quoted)
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

func parseShell(data string) (*SCloudConfig, error) {
	info := strings.Split(data, "\n")
	ret := &SCloudConfig{
		Runcmd:    []string{},
		SshPwauth: SSH_PASSWORD_AUTH_ON,
	}
	for _, cmd := range info {
		if strings.HasPrefix(cmd, "#") || len(strings.Trim(cmd, "")) == 0 {
			continue
		}
		ret.Runcmd = append(ret.Runcmd, cmd)
	}
	return ret, nil
}

func ParseUserData(data string) (*SCloudConfig, error) {
	if !strings.HasPrefix(data, CLOUD_CONFIG_HEADER) {
		return parseShell(data)
	}
	jsonConf, err := jsonutils.ParseYAML(data)
	if err != nil {
		return nil, errors.Wrapf(err, "ParseYAML")
	}
	jsonDict, ok := jsonConf.(*jsonutils.JSONDict)
	if !ok {
		// Anything that is valid YAML but not a mapping, e.g. a list or a
		// scalar, is not a usable cloud-config document.
		return nil, errors.Wrapf(errors.ErrInvalidFormat,
			"cloud-config must be a YAML mapping, got %s", jsonConf.String())
	}
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
