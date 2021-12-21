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
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func MountFusefs(fetcherfsPath, url, tmpdir, token, mntpath string, blocksize int) error {
	var metaPath = path.Join(mntpath, "meta")
	if f, err := os.OpenFile(metaPath, os.O_RDONLY, 0644); err == nil {
		f.Close()
		log.Infof("Already mounted: %s, skip...", mntpath)
		return nil
	}

	// is mounted
	if err := procutils.NewCommand("mountpoint", mntpath).Run(); err == nil {
		out, err := procutils.NewCommand("umount", mntpath).Output()
		if err != nil {
			return errors.Wrapf(err, "umount %s failed: %s", mntpath, out)
		}
	}

	if !fileutils2.Exists(tmpdir) {
		if out, err := procutils.NewCommand("mkdir", "-p", tmpdir).Output(); err != nil {
			return errors.Wrapf(err, "mkdir %s failed: %s", tmpdir, out)
		}
	}

	if !fileutils2.Exists(mntpath) {
		if out, err := procutils.NewCommand("mkdir", "-p", mntpath).Output(); err != nil {
			return errors.Wrapf(err, "mkdir %s failed: %s", mntpath, out)
		}
	}

	var cmd = []string{
		fetcherfsPath,
		"--url", url,
		"--token", token,
		"--tmpdir", tmpdir,
		"--blocksize", strconv.Itoa(blocksize),
		"--mount-point", mntpath,
	}
	log.Infof("%s", strings.Join(cmd, " "))
	out, err := procutils.NewRemoteCommandAsFarAsPossible(cmd[0], cmd[1:]...).Output()
	if err != nil {
		log.Errorf("%v Mount fetcherfs filed: %s %s", cmd, err, out)
		out2, err2 := procutils.NewCommand("umount", mntpath).Output()
		if err2 != nil {
			log.Errorf("umount fetcherfs failed %s %s", err2, out2)
		}
		return errors.Wrapf(err, "mount fetcherfs failed: %s", out)
	}

	var mounted = false
	for i := 0; i < 3; i++ {
		time.Sleep(1 * time.Second)
		if f, err := os.OpenFile(metaPath, os.O_RDONLY, 0644); err == nil {
			f.Close()
			mounted = true
			break
		} else {
			log.Warningf("failed open metaPath %s: %s", metaPath, err)
		}
	}
	if !mounted {
		out2, err2 := procutils.NewCommand("umount", mntpath).Output()
		if err2 != nil {
			log.Errorf("umount fetcherfs failed %s %s", err2, out2)
		}
		return errors.Error("failed open metaPath")
	} else {
		return nil
	}
}
