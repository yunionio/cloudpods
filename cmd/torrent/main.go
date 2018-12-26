package main

import (
	"fmt"
	"os"
	"path/filepath"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"

	"yunion.io/x/log"
	"yunion.io/x/structarg"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/onecloud/pkg/util/torrentutils"
)

type Options struct {
	structarg.BaseOptions

	ROOT    string `help:"Root directory to seed files"`
	TORRENT string `help:"path to torrent file"`

	Tracker []string `help:"Tracker urls, e.g. http://10.168.222.252:6969/announce or udp://tracker.istole.it:6969"`
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

	if len(options.Tracker) > 0 {
		// server mode
		mi, err = torrentutils.GenerateTorrent(root, options.Tracker, options.TORRENT)
		if err != nil {
			log.Fatalf("fail to save torrent file %s", err)
		}

	} else {
		// client mode, load mi from torrent file
		mi, err = metainfo.LoadFromFile(options.TORRENT)
		if err != nil {
			log.Fatalf("fail to open torrent file %s", err)
		}
	}

	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.Debug = false
	clientConfig.Seed = true
	if len(options.Tracker) > 0 {
		// server mode
		clientConfig.DataDir = path.Dir(root)
	} else {
		// client mode
		clientConfig.DataDir = root
	}
	clientConfig.DisableTrackers = false
	clientConfig.DisablePEX = false
	clientConfig.NoDHT = true

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

	go func() {
		<-t.GotInfo()

		files := t.Info().Files
		log.Debugf("Got Info, start download %d files", len(files))
		for i := 0; i < len(files); i += 1 {
			log.Debugf("%d: %s", i, files[i].Path)
		}

		t.DownloadAll()
	}()

	go func() {
		<-client.Closed()
		log.Debugf("client closed, exit!")

		os.Exit(0)
	}()

	for {
		if t.BytesCompleted() == t.Info().TotalLength() {
			fmt.Printf("\rSeeding.............")
		} else {
			fmt.Printf("\rDownload: %.1f%%", float64(t.BytesCompleted())*100.0/float64(t.Info().TotalLength()))
		}
		time.Sleep(time.Second)
	}
}
