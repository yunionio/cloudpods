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

package torrent

import (
	"fmt"
	"os"
	"time"

	"yunion.io/x/log"

	identity_apis "yunion.io/x/onecloud/pkg/apis/identity"
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

type torrentTask struct {
	torrentpath string
	imageId     string
	format      string
}

func (t *torrentTask) Run() {
	log.Infof("Start seed %s ...", t.torrentpath)
	err := seedTorrent(t.torrentpath, t.imageId, t.format)
	if err == nil {
		time.Sleep(10 * time.Second)
	}
}

func (t *torrentTask) Dump() string {
	return fmt.Sprintf("torrentpath: %s imageId: %s.%s", t.torrentpath, t.imageId, t.format)
}

func SeedTorrent(torrentpath string, imageId, format string) error {
	task := &torrentTask{
		torrentpath: torrentpath,
		imageId:     imageId,
		format:      format,
	}
	seedTaskWorkerMan.Run(task, nil, nil)
	return nil
}

func seedTorrent(torrentpath string, imageId, format string) error {
	url, err := auth.GetServiceURL("image", options.Options.Region, "", identity_apis.EndpointInterfacePublic)
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
