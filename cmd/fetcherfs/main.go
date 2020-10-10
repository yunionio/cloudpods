package main

import (
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sevlyar/go-daemon"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/signalutils"
)

func main() {
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

	fetcherFs, err := initFetcherFs()
	if err != nil {
		log.Fatalln(err)
	}
	defer destoryInitFetcherFs()
	c, err := fuse.Mount(
		opt.MountPoint,
		fuse.FSName("fetcherfs"),
		fuse.Subtype("fetcher"),
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
