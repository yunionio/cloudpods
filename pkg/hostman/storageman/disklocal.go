package storageman

import (
	"fmt"
	"os"
	"path"

	"yunion.io/x/log"
)

var _ALTER_SUFFIX_ = ".alter"

type SLocalDisk struct {
	SBaseDisk
	isAlter bool
}

func NewLocalDisk(storage IStorage, id string) *SLocalDisk {
	var ret = new(SLocalDisk)
	ret.SBaseDisk = *NewBaseDisk(storage, id)
	return ret
}

func (d *SLocalDisk) getPath() string {
	return path.Join(d.Storage.GetPath(), d.Id)
}

func (d *SLocalDisk) getAlterPath() string {
	return path.Join(d.Storage.GetPath(), d.Id, _ALTER_SUFFIX_)
}

func (d *SLocalDisk) GetPath() string {
	if d.isAlter {
		return d.getAlterPath()
	} else {
		return d.getPath()
	}
}

func (d *SLocalDisk) Probe() error {
	if _, err := os.Stat(d.getPath()); !os.IsNotExist(err) {
		d.isAlter = false
		return nil
	} else if _, err := os.Stat(d.getAlterPath()); !os.IsNotExist(err) {
		d.isAlter = true
		return nil
	}
	return fmt.Errorf("Disk not found")
}

func (d *SLocalDisk) Delete() error {
	dpath := d.GetPath()
	log.Infof("Delete guest disk %s", dpath)
	if err := d.Storage.DeleteDiskfile(dpath); err != nil {
		return err
	}
	// TODO: PostCreateFromImageFuse umount fuse fs
	d.UmountImageFuse()
	/* ????????????????
	   files = os.listdir(self.storage.path)
	   for f in files:
	       if f.startswith(self.id):
	           if not re.match(r'[a-z0-9\-]*\.\d{14}', f):
	               path = os.path.join(self.storage.path, f)
	               print 'delete backing-file:', path
	               self.storage.delete_diskfile(path)
	*/
	d.Storage.RemoveDisk(d)
	return nil
}
