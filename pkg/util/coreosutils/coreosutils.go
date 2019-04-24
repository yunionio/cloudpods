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

package coreosutils

import (
	"encoding/base64"
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"yunion.io/x/pkg/utils"
)

type SUnitDropins struct {
	Name    string `yaml:"name,omitempty"`
	Content string `yaml:"content,omitempty"`
}

type SUnits struct {
	Name    string        `yaml:"name,omitempty"`
	Mask    *bool         `yaml:"mask,omitempty"`
	Enable  *bool         `yaml:"enable,omitempty"`
	Runtime *bool         `yaml:"runtime,omitempty"`
	Command string        `yaml:"command,omitempty"`
	Content string        `yaml:"content,omitempty"`
	dropIns *SUnitDropins `yaml:"drop_ins,omitempty"`
}

type SUser struct {
	Name              string   `yaml:"name,omitempty"`
	Passwd            string   `yaml:"passwd,omitempty"`
	SshAuthorizedKeys []string `yaml:"ssh_authorized_keys,omitempty"`
}

func NewUser(name, passwd string, pubkeys []string, nohash bool) SUser {
	if !nohash {
		// TODO: replace with crypt
		passwd, _ = utils.EncryptAESBase64("$6$SALT$", passwd)
	}
	return SUser{
		Name:              name,
		Passwd:            passwd,
		SshAuthorizedKeys: pubkeys,
	}
}

type SWriteFile struct {
	Path        string `yaml:"path,omitempty"`
	Content     string `yaml:"content,omitempty"`
	Permissions string `yaml:"permissions,omitempty"`
	Owner       string `yaml:"owner,omitempty"`
	Encoding    string `yaml:"encoding,omitempty"`
}

func NewWriteFile(spath, content, perm, owner string, isbase64 bool) SWriteFile {
	res := SWriteFile{}
	if isbase64 {
		res.Encoding = "base64"
		res.Content = base64.StdEncoding.EncodeToString([]byte(content))
	} else {
		res.Content = content
	}
	res.Path = spath
	res.Permissions = perm
	res.Owner = owner
	return res
}

type SCloudConfig struct {
	Hostname       string                 `yaml:"hostname,omitempty"`
	Users          []SUser                `yaml:"users,omitempty"`
	Coreos         map[string]interface{} `yaml:"coreos,omitempty"`
	WriteFiles     []SWriteFile           `yaml:"write_files,omitempty"`
	ManageEtcHosts string                 `yaml:"manage_etc_hosts,omitempty"`
}

func NewCloudConfig() *SCloudConfig {
	res := new(SCloudConfig)
	res.Users = make([]SUser, 0)
	res.Coreos = map[string]interface{}{"units": []SUnits{}}
	res.WriteFiles = make([]SWriteFile, 0)
	return res
}

func (c *SCloudConfig) SetHostname(hn string) {
	c.Hostname = hn
}

func (c *SCloudConfig) SetEtcHosts(line string) {
	c.ManageEtcHosts = line
}

func (c *SCloudConfig) AddUser(name, passwd string, pubkeys []string, nohash bool) {
	c.Users = append(c.Users, NewUser(name, passwd, pubkeys, nohash))
}

func (c *SCloudConfig) HasUser(name string) bool {
	for _, u := range c.Users {
		if u.Name == name {
			return true
		}
	}
	return false
}

func (c *SCloudConfig) AddWriteFile(spath, content, prem, owner string, base64 bool) {
	if len(prem) == 0 {
		prem = "0644"
	}
	if len(owner) == 0 {
		owner = "root"
	}
	c.WriteFiles = append(c.WriteFiles, NewWriteFile(spath, content, prem, owner, base64))
}

func (c *SCloudConfig) HasWriteFile(spath string) bool {
	for _, f := range c.WriteFiles {
		if f.Path == spath {
			return true
		}
	}
	return false
}

func (c *SCloudConfig) AddUnits(name string, mask, enable, runtime *bool, content, command string, dropins *SUnitDropins) {
	u := SUnits{
		Name:    name,
		Mask:    mask,
		Enable:  enable,
		Runtime: runtime,
		Content: content,
		Command: command,
		dropIns: dropins,
	}
	units := c.Coreos["units"].([]SUnits)
	units = append(units, u)
	c.Coreos["units"] = units
}

func (c *SCloudConfig) AddSwap(dev string) {
	name := fmt.Sprintf("%s.swap", strings.Replace(dev[1:], "/", "-", -1))
	cont := "[Unit]\n"
	cont += fmt.Sprintf("Description=Enable swap on %s\n", dev)
	cont += "[Swap]\n"
	cont += fmt.Sprintf("What=%s\n", dev)
	c.AddUnits(name, nil, nil, nil, cont, "start", nil)
}

func (c *SCloudConfig) AddPartition(dev, mtpath, fs string) {
	name := fmt.Sprintf("%s.mount", strings.Replace(mtpath[1:], "/", "-", -1))
	cont := "[Unit]\n"
	cont += fmt.Sprintf("Description=Mount %s on %s\n", dev, mtpath)
	cont += "[Mount]\n"
	cont += fmt.Sprintf("What=%s\n", dev)
	cont += fmt.Sprintf("Where=%s\n", mtpath)
	cont += fmt.Sprintf("Type=%s\n", fs)
	c.AddUnits(name, nil, nil, nil, cont, "start", nil)
}

func (c *SCloudConfig) SetTimezone(tz string) {
	name := "settimezone.service"
	cont := "[Unit]\n"
	cont += "Description=Set the timezone\n"
	cont += "[Service]\n"
	cont += fmt.Sprintf("ExecStart=/usr/bin/timedatectl set-timezone %s\n", tz)
	cont += "RemainAfterExit=yes\n"
	cont += "Type=oneshot\n"
	c.AddUnits(name, nil, nil, nil, cont, "start", nil)
	conf := ""
	for i := 0; i < 4; i++ {
		conf += fmt.Sprintf("server %d.pool.ntp.org\n", i)
	}
	conf += "restrict default nomodify nopeer noquery limited kod\n"
	conf += "restrict 127.0.0.1\n"
	conf += "restrict [::1]\n"
	c.AddWriteFile("/etc/ntp.conf", conf, "", "", false)
}

func (c *SCloudConfig) AddConfig(name, cfg string) {
	c.Coreos[name] = cfg
}

func (c *SCloudConfig) YunionInit() {
	VERSION := "0.0.2"
	cont := "id: yunion\n"
	cont += "name: Yunion Yun\n"
	cont += fmt.Sprintf("version-id: %s\n", VERSION)
	cont += "home-url: https://yunionyun.com/\n"
	c.AddConfig("oem", cont)
	mark := true
	c.AddUnits("user-configdrive.service", &mark, nil, nil, "", "", nil)
	c.AddUnits("user-configvirtfs.service", &mark, nil, nil, "", "", nil)
}

func (c *SCloudConfig) String() string {
	ys, _ := yaml.Marshal(c)
	return "#cloud-config\n\n" + string(ys)
}

// func (c *SCloudConfig) String() string {
// 	ret, err := yaml.Marshal(c)
// }
