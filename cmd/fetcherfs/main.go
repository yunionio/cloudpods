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
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sevlyar/go-daemon"

	"yunion.io/x/log"
	"yunion.io/x/log/hooks"
	"yunion.io/x/pkg/util/signalutils"
)

func main() {
	if !opt.Foreground {
		cntxt := &daemon.Context{
			WorkDir: "./",
			Umask:   027,
		}

		d, err := cntxt.Reborn()
		if err != nil {
			log.Fatalf("Unable to run: %s", err)
		}
		if d != nil {
			return
		}
		defer cntxt.Release()
	}

	if opt.Debug {
		logFileHook := hooks.LogFileRotateHook{
			LogFileHook: hooks.LogFileHook{
				FileDir:  "/tmp",
				FileName: "fetcherfs.log",
			},
			RotateNum:  10,
			RotateSize: 4096,
		}
		logFileHook.Init()
		defer logFileHook.DeInit()
		log.Logger().AddHook(&logFileHook)
	}

	fetcherFs, err := initFetcherFs()
	if err != nil {
		log.Fatalln(err)
	}
	defer destoryInitFetcherFs()
	c, err := fuse.Mount(
		opt.MountPoint,
		fuse.FSName("fetcherfs"),
		fuse.Subtype("fetcher"),
		// https://github.com/bazil/fuse/issues/175
		// fuse.MaxReadahead(128*1024),
		fuse.MaxReadahead(uint32(opt.Blocksize*1024*1024)),
	)
	if err != nil {
		log.Fatalln(err)
	}
	defer c.Close()

	err = fs.Serve(c, *fetcherFs)
	if err != nil {
		log.Errorf("serve failed %s", err)
	}
}

func init() {
	signalutils.RegisterSignal(func() {
		destoryInitFetcherFs()
		os.Exit(1)
	}, syscall.SIGTERM, syscall.SIGINT)
	signalutils.StartTrap()
}
