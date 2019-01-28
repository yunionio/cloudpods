package tarutils

import (
	"os"
	"path/filepath"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func TarSparseFile(origin, tar string) error {
	origin, _ = filepath.Abs(origin)
	tar, _ = filepath.Abs(tar)
	workDir := filepath.Dir(origin)
	originFile := filepath.Base(origin)
	if err := os.Chdir(workDir); err != nil {
		log.Errorln(err)
		return err
	}
	_, err := procutils.NewCommand("tar", "-Scf", tar, originFile).Run()
	if err != nil {
		log.Errorln("Tar sparse file error: %s", err)
	}
	return nil
}
