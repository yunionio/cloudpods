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

package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/image/torrent"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
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

	ImageSubformatManager.TableSpec().AddIndex(true, "image_id", "format")
}

type SImageSubformat struct {
	SImagePeripheral

	Format string `width:"20" charset:"ascii" nullable:"true"`

	Size     int64  `nullable:"false"`
	Location string `nullable:"false"`
	Checksum string `width:"32" charset:"ascii" nullable:"true"`
	FastHash string `width:"32" charset:"ascii" nullable:"true"`
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
		if err != sql.ErrNoRows {
			log.Errorf("query subimage fail! %s", err)
		}
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
	if options.Options.EnableTorrentService {
		err = self.seedTorrent(image.Id)
		if err != nil {
			log.Errorf("fail to seed torrent %s", err)
			return err
		}
	}
	// log.Infof("Start seeding...")
	return nil
}

func (self *SImageSubformat) Save(image *SImage) error {
	if self.Status == api.IMAGE_STATUS_ACTIVE {
		return nil
	}
	if self.Status != api.IMAGE_STATUS_QUEUED {
		return nil // httperrors.NewInvalidStatusError("cannot save in status %s", self.Status)
	}
	location := image.GetPath(self.Format)
	_, err := db.Update(self, func() error {
		self.Status = api.IMAGE_STATUS_SAVING
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
	nimg, err := img.Clone(location, qemuimg.String2ImageFormat(self.Format), true)
	if err != nil {
		log.Errorf("img.Clone fail %s", err)
		return err
	}
	checksum, err := fileutils2.MD5(location)
	if err != nil {
		log.Errorf("fileutils2.Md5 fail %s", err)
		return err
	}
	fastHash, err := fileutils2.FastCheckSum(location)
	if err != nil {
		log.Errorf("fileutils2.fastChecksum fail %s", err)
		return err
	}
	_, err = db.Update(self, func() error {
		self.Status = api.IMAGE_STATUS_ACTIVE
		self.Location = fmt.Sprintf("%s%s", LocalFilePrefix, location)
		self.Checksum = checksum
		self.FastHash = fastHash
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
	if self.TorrentStatus == api.IMAGE_STATUS_ACTIVE {
		return nil
	}
	if self.TorrentStatus != api.IMAGE_STATUS_QUEUED {
		return nil // httperrors.NewInvalidStatusError("cannot save torrent in status %s", self.Status)
	}
	imgPath := self.getLocalLocation()
	torrentPath := filepath.Join(options.Options.TorrentStoreDir, fmt.Sprintf("%s.torrent", filepath.Base(imgPath)))
	_, err := db.Update(self, func() error {
		self.TorrentStatus = api.IMAGE_STATUS_SAVING
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
	checksum, err := fileutils2.MD5(torrentPath)
	if err != nil {
		log.Errorf("fileutils2.Md5 fail %s", err)
		return err
	}
	_, err = db.Update(self, func() error {
		self.TorrentStatus = api.IMAGE_STATUS_ACTIVE
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
	if len(self.Location) > len(LocalFilePrefix) {
		return self.Location[len(LocalFilePrefix):]
	}
	return ""
}

func (self *SImageSubformat) getLocalTorrentLocation() string {
	if len(self.TorrentLocation) > len(LocalFilePrefix) {
		return self.TorrentLocation[len(LocalFilePrefix):]
	}
	return ""
}

func (self *SImageSubformat) seedTorrent(imageId string) error {
	file := self.getLocalTorrentLocation()
	log.Debugf("add torrent %s to seed...", file)
	return torrent.SeedTorrent(file, imageId, self.Format)
}

func (self *SImageSubformat) StopTorrent() {
	if len(self.TorrentLocation) > 0 {
		torrent.RemoveTorrent(self.getLocalTorrentLocation())
	}
}

func (self *SImageSubformat) RemoveFiles() error {
	self.StopTorrent()
	location := self.getLocalTorrentLocation()
	if len(location) > 0 && fileutils2.IsFile(location) {
		err := os.Remove(location)
		if err != nil {
			return err
		}
	}
	location = self.getLocalLocation()
	if len(location) > 0 && fileutils2.IsFile(location) {
		err := os.Remove(location)
		if err != nil {
			return err
		}
	}
	return nil
}

type SImageSubformatDetails struct {
	Format string

	Size     int64
	Checksum string
	FastHash string
	Status   string

	TorrentSize     int64
	TorrentChecksum string
	TorrentStatus   string

	TorrentSeeding bool
}

func (self *SImageSubformat) GetDetails() SImageSubformatDetails {
	details := SImageSubformatDetails{}

	details.Format = self.Format
	details.Size = self.Size
	details.Checksum = self.Checksum
	details.FastHash = self.FastHash
	details.Status = self.Status
	details.TorrentSize = self.TorrentSize
	details.TorrentChecksum = self.TorrentChecksum
	details.TorrentStatus = self.TorrentStatus

	filePath := self.getLocalTorrentLocation()
	if len(filePath) > 0 {
		details.TorrentSeeding = torrent.GetTorrentSeeding(filePath)
	}

	return details
}

func (self *SImageSubformat) isActive(useFast bool) bool {
	return isActive(self.getLocalLocation(), self.Size, self.Checksum, self.FastHash, useFast)
}

func (self *SImageSubformat) isTorrentActive() bool {
	return isActive(self.getLocalTorrentLocation(), self.TorrentSize, self.TorrentChecksum, "", false)
}

func (self *SImageSubformat) setStatus(status string) error {
	_, err := db.Update(self, func() error {
		self.Status = status
		return nil
	})
	return err
}

func (self *SImageSubformat) setTorrentStatus(status string) error {
	_, err := db.Update(self, func() error {
		self.TorrentStatus = status
		return nil
	})
	return err
}

func (self *SImageSubformat) checkStatus(useFast bool) {
	if self.isActive(useFast) {
		if self.Status != api.IMAGE_STATUS_ACTIVE {
			self.setStatus(api.IMAGE_STATUS_ACTIVE)
		}
		if len(self.FastHash) == 0 {
			fastHash, err := fileutils2.FastCheckSum(self.getLocalLocation())
			if err != nil {
				log.Errorf("checkStatus fileutils2.FastChecksum fail %s", err)
			} else {
				_, err := db.Update(self, func() error {
					self.FastHash = fastHash
					return nil
				})
				if err != nil {
					log.Errorf("checkStatus save FastHash fail %s", err)
				}
			}
		}
	} else {
		if self.Status != api.IMAGE_STATUS_QUEUED {
			self.setStatus(api.IMAGE_STATUS_QUEUED)
		}
	}
	if self.isTorrentActive() {
		if self.TorrentStatus != api.IMAGE_STATUS_ACTIVE {
			self.setTorrentStatus(api.IMAGE_STATUS_ACTIVE)
		}
	} else {
		if self.TorrentStatus != api.IMAGE_STATUS_QUEUED {
			self.setTorrentStatus(api.IMAGE_STATUS_QUEUED)
		}
	}
}

func (self *SImageSubformat) SetStatusSeeding(seeding bool) {
	filePath := self.getLocalTorrentLocation()
	if len(filePath) > 0 {
		torrent.SetTorrentSeeding(filePath, seeding)
	}
}
