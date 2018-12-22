package torrent

import (
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"fmt"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/image/options"
)

var (
	torrentClient *torrent.Client
	torrentTable  = make(map[string]*torrent.Torrent)
)

func GetTrackers() []string {
	urls, err := auth.GetServiceURLs("torrent-tracker", options.Options.Region, "", "")
	if err != nil {
		log.Errorf("fail to get torrent-tracker")
		return nil
	}
	return urls
}

func InitTorrentClient() error {
	urls := GetTrackers()
	if len(urls) == 0 {
		log.Errorf("no valid torrent-tracker")
		return fmt.Errorf("no valid torrent-tracker")
	}

	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.Debug = false
	clientConfig.Seed = true
	clientConfig.DataDir = options.Options.FilesystemStoreDatadir
	clientConfig.DisableTrackers = false
	clientConfig.DisablePEX = false
	clientConfig.NoDHT = true

	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		log.Errorf("error creating client: %s", err)
		return err
	}
	torrentClient = client

	return nil
}

func CloseTorrentClient() {
	if torrentClient != nil {
		torrentClient.Close()
	}
}

func AddTorrent(filepath string) error {
	mi, err := metainfo.LoadFromFile(filepath)
	if err != nil {
		log.Errorf("fail to open torrent file %s", err)
		return err
	}
	t, err := torrentClient.AddTorrent(mi)
	if err != nil {
		log.Errorf("AddTorrent fail %s", err)
		return err
	}

	torrentTable[filepath] = t

	go func() {
		<-t.GotInfo()
		t.DownloadAll()
	}()

	return nil
}

func RemoveTorrent(filepath string) {
	if t, ok := torrentTable[filepath]; ok {
		t.Drop()
		delete(torrentTable, filepath)
	}
}
