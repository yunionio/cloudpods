package torrentutils

import (
	"os"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"

	"yunion.io/x/log"

)

func GenerateTorrent(root string, trackers []string, torrentFile string) (*metainfo.MetaInfo, error) {
	log.Debugf("generating torrent file ...")

	os.Remove(torrentFile)

	mi := metainfo.MetaInfo{
		AnnounceList: [][]string{},
	}
	for _, a := range trackers {
		mi.AnnounceList = append(mi.AnnounceList, []string{a})
	}
	mi.SetDefaults()
	isPrivate := true
	info := metainfo.Info{
		PieceLength: 2 * 1024 * 1024,
		Private:     &isPrivate,
	}
	err := info.BuildFromFilePath(root)
	if err != nil {
		return nil, err
	}
	mi.InfoBytes, err = bencode.Marshal(info)
	if err != nil {
		return nil, err
	}

	torrentFp, err := os.Create(torrentFile)
	if err != nil {
		log.Fatalf("fail to create torrent file %s", err)
	}
	defer torrentFp.Close()

	err = mi.Write(torrentFp)
	if err != nil {
		return nil, err
	}
	log.Debugf("generating torrent file %s ...done!", torrentFile)
	return &mi, nil
}

