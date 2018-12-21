package storageman

import "os"

type SLocalDisk struct {
	*SBaseDisk
}

func NewLocalDisk(storage IStorage, id string) *SLocalDisk {
	var ret = new(SLocalDisk)
	ret.SBaseDisk = NewBaseDisk(storage, id)
	return ret
}

func (d *SLocalDisk) GetId() string {
	return d.Id
}

func (d *SLocalDisk) Probe() bool {
	if _, err := os.Stat(d.getPath()); !os.IsNotExist(err) {
		return true
	}
	// TODO alter ??
	return false
}
