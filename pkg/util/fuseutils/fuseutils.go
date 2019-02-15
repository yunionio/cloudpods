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
	if _, err := procutils.NewCommand("mountpoint", mntpath).Run(); err == nil {
		procutils.NewCommand("umount", mntpath).Run()
	}

	if !fileutils2.Exists(tmpdir) {
		if _, err := procutils.NewCommand("mkdir", "-p", tmpdir).Run(); err != nil {
			return err
		}
	}

	if !fileutils2.Exists(mntpath) {
		if _, err := procutils.NewCommand("mkdir", "-p", mntpath).Run(); err != nil {
			return err
		}
	}

	var opts = fmt.Sprintf("url=%s", url)
	opts += fmt.Sprintf(",tmpdir=%s", tmpdir)
	opts += fmt.Sprintf(",token=%s", token)
	opts += fmt.Sprintf(",blocksize=%d", blocksize)

	var cmd = []string{fetcherfsPath, "-s", "-p", opts, mntpath}
	log.Infof("%s", strings.Join(cmd, " "))
	_, err := procutils.NewCommand(cmd[0], cmd[1:]...).Run()
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
