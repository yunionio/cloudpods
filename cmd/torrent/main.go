package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/structarg"

	"github.com/anacrolix/torrent/storage"
	"io/ioutil"
	"net/http"
	"yunion.io/x/onecloud/pkg/util/nodeid"
	"yunion.io/x/onecloud/pkg/util/torrentutils"
)

type Options struct {
	structarg.BaseOptions

	ROOT    string `help:"Root directory to seed files"`
	TORRENT string `help:"path to torrent file"`

	Tracker []string `help:"Tracker urls, e.g. http://10.168.222.252:6969/announce or udp://tracker.istole.it:6969"`

	Debug bool `help:"turn on debug"`

	CallbackURL string `help:"callback notification URL"`
}

func exitSignalHandlers(client *torrent.Client) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	for {
		log.Printf("close signal received: %+v", <-c)
		client.Close()
	}
}

func main() {
	options := Options{}

	parser, err := structarg.NewArgumentParser(&options, "torrent-srv", "bit-torrent server", "2018")
	if err != nil {
		log.Fatalf("%s", err)
	}

	parser.ParseArgs(os.Args[1:], true)

	if options.Help {
		fmt.Println(parser.HelpString())
		return
	}

	if len(os.Args) <= 1 {
		fmt.Print(parser.Usage())
		return
	}

	if options.Version {
		fmt.Println(version.GetJsonString())
		return
	}

	root, err := filepath.Abs(options.ROOT)
	if err != nil {
		log.Fatalf("fail to get absolute path: %s", err)
	}

	var mi *metainfo.MetaInfo
	var rootDir string

	if len(options.Tracker) > 0 {
		// server mode
		mi, err = torrentutils.GenerateTorrent(root, options.Tracker, options.TORRENT)
		if err != nil {
			log.Fatalf("fail to save torrent file %s", err)
		}
		rootDir = filepath.Dir(root)

	} else {
		// client mode, load mi from torrent file
		mi, err = metainfo.LoadFromFile(options.TORRENT)
		if err != nil {
			log.Fatalf("fail to open torrent file %s", err)
		}
		rootDir = root
	}

	nodeId, err := nodeid.GetNodeId()
	if err != nil {
		log.Errorf("fail to generate node id: %s", err)
		return
	}

	log.Infof("Set torrent server as node %s", nodeId)

	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.PeerID = nodeId[:20]
	clientConfig.Debug = options.Debug
	clientConfig.Seed = true
	clientConfig.NoUpload = false

	info, err := mi.UnmarshalInfo()
	if err != nil {
		log.Errorf("fail to unmarshalinfo %s", err)
		return
	}
	log.Infof("To download file %s", info.Name)
	tmpDir := filepath.Join(rootDir, fmt.Sprintf("%s%s", info.Name, ".tmp"))

	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	clientConfig.DefaultStorage = storage.NewFileWithCustomPathMaker(tmpDir,
		func(baseDir string, info *metainfo.Info, infoHash metainfo.Hash) string {
			return filepath.Dir(baseDir)
		},
	)

	clientConfig.DisableTrackers = false
	clientConfig.DisablePEX = true
	clientConfig.NoDHT = true
	clientConfig.NominalDialTimeout = 1 * time.Second
	clientConfig.MinDialTimeout = 100 * time.Millisecond
	clientConfig.HandshakesTimeout = 100 * time.Second

	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		log.Fatalf("error creating client: %s", err)
	}
	defer client.Close()

	go exitSignalHandlers(client)

	t, err := client.AddTorrent(mi)
	if err != nil {
		log.Fatalf("%s", err)
	}

	<-t.GotInfo()
	t.DownloadAll()

	stop := false

	go func() {
		<-client.Closed()
		log.Debugf("client closed, exit!")

		stop = true
	}()

	for !stop {
		if t.BytesCompleted() == t.Info().TotalLength() {
			fmt.Printf("\rSeeding.............")
			if len(options.CallbackURL) > 0 {
				for tried := 0; tried < 10; tried += 1 {
					resp, err := http.Get(options.CallbackURL)
					if err == nil && resp.StatusCode < 300 {
						break
					}
					if err != nil {
						log.Errorf("callback fail %s", err)
					} else {
						respBody, _ := ioutil.ReadAll(resp.Body)
						log.Errorf("callback response error %s", string(respBody))
					}
					time.Sleep(time.Duration(tried+1) * 10 * time.Second)
				}
			}
		} else {
			fmt.Printf("\rDownload: %.1f%%", float64(t.BytesCompleted())*100.0/float64(t.Info().TotalLength()))
		}
		// client.WriteStatus(os.Stdout)
		time.Sleep(time.Second)
	}
}
