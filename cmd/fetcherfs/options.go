package main

import (
	"os"

	"yunion.io/x/log"
	"yunion.io/x/structarg"
)

// /opt/yunion/fetchclient/bin/fetcherfs -s -o url=http://10.168.222.178:48885/snapshots/a6ca006d-d246-4a52-8aea-c581bb1ab7ce/f746ab85-3a29-4139-8bd7-a661f14607ac,tmpdir=/tmpxx/fusetmp,token=gAAAAABfGod6E1i_SQHzTCU6BOJx8dNrGaqaEKhg7uc9j3wjuxGs_bKE7esneHo4HvV23h0Rq3ntVIHlgcPG9HBVsTXGWJ4M3cT3Qnig_RMUxLw83BjAszwd0mwkhz-Uux3smcCVYNgy98jXIe2XhA0_3mkMYkImF-M9OAOuyTBAo81Gbc6HXqaLQP0Tmd9zu4S7FuzUBdzT,blocksize=8 /tmpxx/fusemnt/dc2ad406-b989-4fcf-8e07-743a310d8b85

const BLOCK_SIZE = 8

type Options struct {
	Url        string `help:"destination to fetch content" required:"true"`
	Tmpdir     string `help:"temporary dir save fs content" required:"true"`
	Token      string `help:"authentication information to access given url" required:"true"`
	Blocksize  int    `help:"block size of content file system(MB)"`
	MountPoint string `help:"mount path of fuse fs" required:"true"`
	Debug      bool   `help:"enable debug go fuse"`
}

var opt = &Options{}

func init() {
	// structarg.NewArgumentParser(&BaseOptions{}
	parser, err := structarg.NewArgumentParser(opt, "", "", "")
	if err != nil {
		log.Fatalf("Error define argument parser: %v", err)
	}
	err = parser.ParseArgs2(os.Args[1:], true, true)
	if err != nil {
		log.Fatalf("Failed parse args %s", err)
	}
	log.Errorf("%v", opt)
	if opt.Blocksize <= 0 {
		opt.Blocksize = BLOCK_SIZE
	}
}
