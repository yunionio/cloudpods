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

package fuseutils

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const DEFAULT_BLOCKSIZE = 8

func MountFusefs(fetcherfsPath, url, tmpdir, token, mntpath string, blocksize int) error {
	var metaPath = path.Join(mntpath, "meta")
	if f, err := os.OpenFile(metaPath, os.O_RDONLY, 0644); err == nil {
		f.Close()
		log.Infof("Already mounted: %s, skip...", mntpath)
		return nil
	}

	// is mounted
	if err := procutils.NewCommand("mountpoint", mntpath).Run(); err == nil {
		procutils.NewCommand("umount", mntpath).Output()
	}

	if !fileutils2.Exists(tmpdir) {
		if err := procutils.NewCommand("mkdir", "-p", tmpdir).Run(); err != nil {
			return err
		}
	}

	if !fileutils2.Exists(mntpath) {
		if err := procutils.NewCommand("mkdir", "-p", mntpath).Run(); err != nil {
			return err
		}
	}

	var opts = fmt.Sprintf("url=%s", url)
	opts += fmt.Sprintf(",tmpdir=%s", tmpdir)
	opts += fmt.Sprintf(",token=%s", token)
	opts += fmt.Sprintf(",blocksize=%d", blocksize)

	var cmd = []string{fetcherfsPath, "-s", "-o", opts, mntpath}
	log.Infof("%s", strings.Join(cmd, " "))
	err := procutils.NewCommand(cmd[0], cmd[1:]...).Run()
	if err != nil {
		log.Errorf("Mount fetcherfs filed: %s", err)
		procutils.NewCommand("umount", mntpath).Run()
		return err
	}

	time.Sleep(200 * time.Millisecond)
	if f, err := os.OpenFile(metaPath, os.O_RDONLY, 0644); err == nil {
		f.Close()
		return nil
	} else {
		log.Errorln(err)
		return err
	}
}
