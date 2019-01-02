package fuseutils

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
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
	if err := exec.Command("mountpoint", mntpath).Run(); err == nil {
		exec.Command("umount", mntpath).Run()
	}

	if !fileutils2.Exists(tmpdir) {
		if err := exec.Command("mkdir", "-p", tmpdir).Run(); err != nil {
			return err
		}
	}

	if !fileutils2.Exists(mntpath) {
		if err := exec.Command("mkdir", "-p", mntpath).Run(); err != nil {
			return err
		}
	}

	var opts = fmt.Sprintf("url=%s", url)
	opts += fmt.Sprintf(",tmpdir=%s", tmpdir)
	opts += fmt.Sprintf(",token=%s", token)
	opts += fmt.Sprintf(",blocksize=%d", blocksize)

	var cmd = []string{fetcherfsPath, "-s", "-p", opts, mntpath}
	log.Infof("%s", strings.Join(cmd, " "))
	err := exec.Command(cmd[0], cmd[1:]...).Run()
	if err != nil {
		log.Errorf("Mount fetcherfs filed: %s", err)
		exec.Command("umount", mntpath).Run()
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
