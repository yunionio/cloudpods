package torrent

import (
	"context"
	"fmt"
	"net/http"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/nodeid"
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

	nodeId, err := nodeid.GetNodeId()
	if err != nil {
		log.Errorf("fail to generate node id: %s", err)
		return err
	}

	log.Infof("Set torrent server as node %s", nodeId)

	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.PeerID = nodeId[:20]
	clientConfig.Debug = false
	clientConfig.Seed = true
	clientConfig.NoUpload = false
	clientConfig.DataDir = options.Options.FilesystemStoreDatadir
	clientConfig.DisableTrackers = false
	clientConfig.DisablePEX = true
	clientConfig.NoDHT = true

	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		log.Errorf("error creating client: %s", err)
		return err
	}
	torrentClient = client

	log.Infof("torrent client initialized")

	return nil
}

func InitTorrentHandler(app *appsrv.Application) {
	app.AddDefaultHandler("GET", "/torrent_stats", TorrentStatsHandler, "torrent_stats")
}

func TorrentStatsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if torrentClient != nil {
		torrentClient.WriteStatus(w)
	}
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

func GetTorrent(filepath string) *torrent.Torrent {
	if t, ok := torrentTable[filepath]; ok {
		return t
	}
	return nil
}

func RemoveTorrent(filepath string) {
	if t, ok := torrentTable[filepath]; ok {
		t.Drop()
		delete(torrentTable, filepath)
	}
}
