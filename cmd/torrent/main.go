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

package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/util/atexit"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/nodeid"
	"yunion.io/x/onecloud/pkg/util/torrentutils"
)

type Options struct {
	structarg.BaseOptions

	ROOT    string `help:"Root directory to seed files"`
	TORRENT string `help:"path to torrent file"`

	Tracker []string `help:"Tracker urls, e.g. http://10.168.222.252:6969/announce or udp://tracker.istole.it:6969"`

	Debug   bool `help:"turn on debug" default:"false"`
	Verbose bool `help:"verbose mode" default:"false"`

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
	defer atexit.Handle()

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

	if len(options.Tracker) > 0 && !fileutils2.Exists(options.TORRENT) {
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

	info, err := mi.UnmarshalInfo()
	if err != nil {
		log.Errorf("fail to unmarshalinfo %s", err)
		return
	}

	hasher := sha1.New()

	nodeId, err := nodeid.GetNodeId()
	if err != nil {
		log.Errorf("fail to generate node id: %s", err)
		return
	}

	hasher.Write(nodeId)
	hasher.Write(info.Pieces)

	peerIdStr := fmt.Sprintf("%x", hasher.Sum(nil))

	log.Infof("Set torrent server as node %s", peerIdStr[:20])

	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.PeerID = peerIdStr[:20]
	clientConfig.Debug = options.Debug
	clientConfig.Seed = true
	clientConfig.NoUpload = false

	log.Infof("To sync torrent files for %s", info.Name)
	tmpDir := filepath.Join(rootDir, fmt.Sprintf("%s%s", info.Name, ".tmp"))

	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0700)
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

	start := time.Now()

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

	finish := false

	for !stop {
		if t.BytesCompleted() == t.Info().TotalLength() {
			if !finish {
				finish = true
				fmt.Printf("Download complete, takes %d seconds\n", time.Now().Sub(start)/time.Second)
				if len(options.CallbackURL) > 0 {
					maxTried := 10
					for tried := 0; tried < maxTried; tried += 1 {
						resp, err := http.Post(options.CallbackURL, "", nil)
						if err == nil && resp.StatusCode < 300 {
							break
						}
						if err != nil {
							log.Errorf("callback fail %s", err)
						} else {
							defer resp.Body.Close()
							respBody, _ := io.ReadAll(resp.Body)
							log.Errorf("callback response error %s", string(respBody))
						}
						time.Sleep(time.Duration(tried+1) * 10 * time.Second)
					}
				}
			}
			fmt.Printf("\rSeeding.............")
		} else {
			fmt.Printf("\rDownload: %.1f%%", float64(t.BytesCompleted())*100.0/float64(t.Info().TotalLength()))
		}
		// client.WriteStatus(os.Stdout)
		time.Sleep(time.Second)
	}
}
