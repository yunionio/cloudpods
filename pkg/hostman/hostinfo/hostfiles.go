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

package hostinfo

import (
	"fmt"
	"os"
	"path/filepath"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/system_service"
	"yunion.io/x/onecloud/pkg/util/apparmorutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func (h *SHostInfo) loadExistingHostFiles() ([]api.SHostFile, error) {
	if !fileutils2.Exists(options.HostOptions.HostFilesPath) {
		return nil, nil
	}

	hostFilesContent, err := os.ReadFile(options.HostOptions.HostFilesPath)
	if err != nil {
		return nil, errors.Wrap(err, "os.ReadFile")
	}
	hostFilesJson, err := jsonutils.Parse(hostFilesContent)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.Parse")
	}

	hostFiles := make([]api.SHostFile, 0)
	err = hostFilesJson.Unmarshal(&hostFiles)
	if err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}

	return hostFiles, nil
}

func (h *SHostInfo) saveHostFiles(hostfiles []api.SHostFile) error {
	hostFilesJson := jsonutils.Marshal(hostfiles)
	err := os.WriteFile(options.HostOptions.HostFilesPath, []byte(hostFilesJson.String()), 0644)
	if err != nil {
		return errors.Wrap(err, "os.WriteFile")
	}
	return nil
}

type hostFilePaire struct {
	old *api.SHostFile
	new *api.SHostFile
}

func (h *SHostInfo) OnHostFilesChanged(hostfiles []api.SHostFile) error {
	existing, err := h.loadExistingHostFiles()
	if err != nil {
		return errors.Wrap(err, "loadExistingHostFiles")
	}
	pairMap := make(map[string]*hostFilePaire)

	for i := range existing {
		hf := fixTelegrafConfPath(&existing[i])
		pairMap[hf.Id] = &hostFilePaire{
			old: hf,
		}
	}
	for i := range hostfiles {
		hf := fixTelegrafConfPath(&hostfiles[i])
		if _, ok := pairMap[hf.Id]; !ok {
			pairMap[hf.Id] = &hostFilePaire{
				new: hf,
			}
		} else {
			pairMap[hf.Id].new = hf
		}
	}

	for _, pair := range pairMap {
		if pair.new == nil {
			// delete file
			err := handleHostFileRemove(pair.old)
			if err != nil {
				return errors.Wrap(err, "handleHostFileRemove")
			}
		} else {
			// update file
			err := handleHostFileChanged(pair)
			if err != nil {
				return errors.Wrap(err, "handleHostFileChanged")
			}
		}
	}

	err = h.saveHostFiles(hostfiles)
	if err != nil {
		return errors.Wrap(err, "saveHostFiles")
	}

	return nil
}

func handleHostFileChanged(pair *hostFilePaire) error {
	switch pair.new.Type {
	case string(api.ApparmorProfile):
		if pair.old == nil || pair.old.Content != pair.new.Content {
			if !apparmorutils.IsEnabled() {
				log.Warningf("apparmor is not enabled, skip loading profile %s", pair.new.Name)
			} else {
				log.Infof("load apparmor profile %s", pair.new.Name)
				err := apparmorutils.Parser(pair.new.Content)
				if err != nil {
					return errors.Wrap(err, "apparmorutils.Parser")
				}
			}
		}
	default:
		changed := false
		if pair.old == nil || pair.old.Path != pair.new.Path || pair.old.Content != pair.new.Content {
			// new or changed file
			log.Infof("update host file %s", pair.new.Name)
			err := procutils.FilePutContents(pair.new.Path, pair.new.Content)
			if err != nil {
				return errors.Wrap(err, "procutils.FilePutContents")
			}
			changed = true
		}
		if pair.old != nil && pair.old.Path != pair.new.Path {
			// remove old file
			log.Infof("remove obsoleted host file %s", pair.old.Name)
			err := procutils.NewRemoteCommandAsFarAsPossible("rm", "-f", pair.old.Path).Run()
			if err != nil {
				return errors.Wrapf(err, "remove file %s", pair.old.Path)
			}
			changed = true
		}
		if changed {
			err := finalizeFileChange(pair.new)
			if err != nil {
				return errors.Wrap(err, "finalizeFileChange")
			}
		}
	}
	return nil
}

func handleHostFileRemove(hostFile *api.SHostFile) error {
	switch hostFile.Type {
	case string(api.ApparmorProfile):
		// do nothing
	default:
		log.Infof("remove file %s", hostFile.Path)
		err := procutils.NewRemoteCommandAsFarAsPossible("rm", "-f", hostFile.Path).Run()
		if err != nil {
			return errors.Wrapf(err, "remove file %s", hostFile.Path)
		}
		{
			err := finalizeFileChange(hostFile)
			if err != nil {
				return errors.Wrap(err, "finalizeFileChange")
			}
		}
	}
	return nil
}

func finalizeFileChange(hostFile *api.SHostFile) error {
	switch hostFile.Type {
	case string(api.TelegrafConf):
		telegrafService := system_service.NewTelegrafService()
		err := telegrafService.ReloadTelegraf()
		if err != nil {
			log.Warningf("failed to reload telegraf: %s", err)
		}
	case string(api.ScriptFile):
		err := procutils.NewRemoteCommandAsFarAsPossible("chmod", "+x", hostFile.Path).Run()
		if err != nil {
			log.Warningf("failed to chmod script file %s: %s", hostFile.Path, err)
		}
	}
	return nil
}

func fixTelegrafConfPath(hostFile *api.SHostFile) *api.SHostFile {
	if hostFile.Type != string(api.TelegrafConf) {
		return hostFile
	}
	var baseFile string
	if len(hostFile.Path) > 0 {
		baseFile = filepath.Base(hostFile.Path)
	}
	if len(baseFile) == 0 {
		baseFile = fmt.Sprintf("%s.conf", hostFile.Name)
	}
	telegrafDDir := system_service.GetTelegrafConfDDir()
	procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", telegrafDDir).Run()
	hostFile.Path = filepath.Join(telegrafDDir, baseFile)
	return hostFile
}
