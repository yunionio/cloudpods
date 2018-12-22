package models

import (
	"fmt"
	"os"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/qemuimg"
	"yunion.io/x/onecloud/pkg/image/torrent"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/torrentutils"
)

type SImageSubformatManager struct {
	db.SResourceBaseManager
}

var ImageSubformatManager *SImageSubformatManager

func init() {
	ImageSubformatManager = &SImageSubformatManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SImageSubformat{},
			"image_subformats",
			"image_subformat",
			"image_subformats",
		),
	}

	ImageSubformatManager.TableSpec().AddIndex(true, "image_id", "format", "is_torrent")
}

type SImageSubformat struct {
	SImagePeripheral

	Format string `width:"20" charset:"ascii" nullable:"true"`

	Size     int64  `nullable:"false"`
	Location string `nullable:"false"`
	Checksum string `width:"32" charset:"ascii" nullable:"true"`
	Status   string `nullable:"false"`

	TorrentSize     int64  `nullable:"false"`
	TorrentLocation string `nullable:"true"`
	TorrentChecksum string `width:"32" charset:"ascii" nullable:"true"`
	TorrentStatus   string `nullable:"false"`
}

func (manager *SImageSubformatManager) FetchSubImage(id string, format string) *SImageSubformat {
	q := manager.Query().Equals("image_id", id).Equals("format", format)
	subImg := SImageSubformat{}
	err := q.First(&subImg)
	if err != nil {
		log.Errorf("query subimage fail!")
		return nil
	}
	return &subImg
}

func (manager *SImageSubformatManager) GetAllSubImages(id string) []SImageSubformat {
	q := manager.Query().Equals("image_id", id)
	var subImgs []SImageSubformat
	err := db.FetchModelObjects(manager, q, &subImgs)
	if err != nil {
		log.Errorf("query subimage fail!")
		return nil
	}
	return subImgs
}

func (self *SImageSubformat) DoConvert(image *SImage) error {
	err := self.Save(image)
	if err != nil {
		log.Errorf("fail to convert image %s", err)
		return err
	}
	err = self.SaveTorrent()
	if err != nil {
		log.Errorf("fail to convert image torrent %s", err)
		return err
	}
	err = self.seedTorrent()
	if err != nil {
		log.Errorf("fail to seed torrent %s", err)
		return err
	}
	return nil
}

func (self *SImageSubformat) Save(image *SImage) error {
	if self.Status == IMAGE_STATUS_ACTIVE {
		return nil
	}
	if self.Status != IMAGE_STATUS_QUEUED {
		return nil // httperrors.NewInvalidStatusError("cannot save in status %s", self.Status)
	}
	location := image.GetPath(self.Format)
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Status = IMAGE_STATUS_SAVING
		self.Location = fmt.Sprintf("%s%s", LocalFilePrefix, location)
		return nil
	})
	if err != nil {
		log.Errorf("updateStatus fail %s", err)
		return err
	}
	img, err := image.getQemuImage()
	if err != nil {
		log.Errorf("image.getQemuImage fail %s", err)
		return err
	}
	nimg, err := img.Clone(location, qemuimg.TImageFormat(self.Format), true)
	if err != nil {
		log.Errorf("img.Clone fail %s", err)
		return err
	}
	checksum, err := fileutils2.Md5(location)
	if err != nil {
		log.Errorf("fileutils2.Md5 fail %s", err)
		return err
	}
	_, err = self.GetModelManager().TableSpec().Update(self, func() error {
		self.Status = IMAGE_STATUS_ACTIVE
		self.Location = fmt.Sprintf("%s%s", LocalFilePrefix, location)
		self.Checksum = checksum
		self.Size = nimg.ActualSizeBytes
		return nil
	})
	if err != nil {
		log.Errorf("updateStatus fail %s", err)
		return err
	}
	return nil
}

func (self *SImageSubformat) SaveTorrent() error {
	if self.TorrentStatus == IMAGE_STATUS_ACTIVE {
		return nil
	}
	if self.TorrentStatus != IMAGE_STATUS_QUEUED {
		return nil // httperrors.NewInvalidStatusError("cannot save torrent in status %s", self.Status)
	}
	imgPath := self.getLocalLocation()
	torrentPath := fmt.Sprintf("%s.torrent", imgPath)
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.TorrentStatus = IMAGE_STATUS_SAVING
		self.TorrentLocation = fmt.Sprintf("%s%s", LocalFilePrefix, torrentPath)
		return nil
	})
	if err != nil {
		log.Errorf("updateStatus fail %s", err)
		return err
	}
	_, err = torrentutils.GenerateTorrent(imgPath, torrent.GetTrackers(), torrentPath)
	if err != nil {
		log.Errorf("torrentutils.GenerateTorrent fail %s", err)
		return err
	}
	checksum, err := fileutils2.Md5(torrentPath)
	if err != nil {
		log.Errorf("fileutils2.Md5 fail %s", err)
		return err
	}
	_, err = self.GetModelManager().TableSpec().Update(self, func() error {
		self.TorrentStatus = IMAGE_STATUS_ACTIVE
		self.TorrentLocation = fmt.Sprintf("%s%s", LocalFilePrefix, torrentPath)
		self.TorrentChecksum = checksum
		self.TorrentSize = fileutils2.FileSize(torrentPath)
		return nil
	})
	if err != nil {
		log.Errorf("updateStatus fail %s", err)
		return err
	}
	return nil
}

func (self *SImageSubformat) getLocalLocation() string {
	return self.Location[len(LocalFilePrefix):]
}

func (self *SImageSubformat) getLocalTorrentLocation() string {
	return self.TorrentLocation[len(LocalFilePrefix):]
}

func (self *SImageSubformat) seedTorrent() error {
	file := self.getLocalTorrentLocation()
	log.Debugf("add torrent %s to seed...", file)
	return torrent.AddTorrent(file)
}

func (self *SImageSubformat) StopTorrent() {
	if len(self.TorrentLocation) > 0 {
		torrent.RemoveTorrent(self.getLocalTorrentLocation())
	}
}

func (self *SImageSubformat) RemoveFiles() error {
	self.StopTorrent()
	if len(self.TorrentLocation) > 0 {
		err := os.Remove(self.getLocalTorrentLocation())
		if err != nil {
			return err
		}
	}
	if len(self.Location) > 0 {
		err := os.Remove(self.getLocalLocation())
		if err != nil {
			return err
		}
	}
	return nil
}
