package torrent

import (
	"fmt"
	"os"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type STorrentProcessState struct {
	process *os.Process
	seeding bool
}

var (
	torrentTable      = make(map[string]*STorrentProcessState)
	seedTaskWorkerMan *appsrv.SWorkerManager
)

const (
	TORRENT_TRACKER_SERVICE = "torrent-tracker"
)

func init() {
	seedTaskWorkerMan = appsrv.NewWorkerManager("seedTaskWorkerManager", 1, 1024, false)
}

func (stat *STorrentProcessState) StopAndWait() error {
	err := stat.process.Kill()
	if err != nil {
		log.Errorf("kill error %s", err)
		return err
	}
	_, err = stat.process.Wait()
	if err != nil {
		log.Errorf("wait error %s", err)
		return err
	}
	return nil
}

func GetTrackers() []string {
	urls, err := auth.GetServiceURLs(TORRENT_TRACKER_SERVICE, options.Options.Region, "", "")
	if err != nil {
		log.Errorf("fail to get torrent-tracker")
		return nil
	}
	return urls
}

func SeedTorrent(torrentpath string, imageId, format string) error {
	seedTaskWorkerMan.Run(func() {
		log.Infof("Start seed %s ...", torrentpath)
		err := seedTorrent(torrentpath, imageId, format)
		if err == nil {
			time.Sleep(10 * time.Second)
		}
	}, nil, nil)
	return nil
}

func seedTorrent(torrentpath string, imageId, format string) error {
	url, err := auth.GetServiceURL("image", options.Options.Region, "", "public")
	if err != nil {
		return err
	}
	args := []string{
		options.Options.TorrentClientPath,
		options.Options.FilesystemStoreDatadir,
		torrentpath,
		"--callback-url",
		fmt.Sprintf("%s/images/%s/update-torrent-status?format=%s", url, imageId, format),
	}
	proc, err := sysutils.Start(false, args...)
	if err != nil {
		return err
	}
	torrentTable[torrentpath] = &STorrentProcessState{
		process: proc,
		seeding: false,
	}
	return nil
}

func SetTorrentSeeding(filepath string, seeding bool) {
	if _, ok := torrentTable[filepath]; ok {
		torrentTable[filepath].seeding = seeding
	}
}

func GetTorrentSeeding(filepath string) bool {
	if t, ok := torrentTable[filepath]; ok {
		return t.seeding
	}
	return false
}

func RemoveTorrent(filepath string) {
	if t, ok := torrentTable[filepath]; ok {
		t.StopAndWait()
		delete(torrentTable, filepath)
	}
}

func StopTorrents() {
	for k := range torrentTable {
		torrentTable[k].StopAndWait()
	}
}
