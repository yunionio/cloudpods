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
	"fmt"
	"time"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

// +onecloud:swagger-gen-ignore
type SGuestFloppyManager struct {
	db.SModelBaseManager
}

var GuestFloppyManager *SGuestFloppyManager

func init() {
	GuestFloppyManager = &SGuestFloppyManager{
		SModelBaseManager: db.NewModelBaseManager(
			SGuestfloppy{},
			"guestfloppy_tbl",
			"guestfloppy",
			"guestfloppys",
		),
	}
	GuestFloppyManager.SetVirtualObject(GuestFloppyManager)
}

// +onecloud:swagger-gen-ignore
type SGuestfloppy struct {
	db.SModelBase

	RowId         int64     `primary:"true" auto_increment:"true" list:"user"`
	Id            string    `width:"36" charset:"ascii" `                 // = Column(VARCHAR(36, charset='ascii'), primary_key=True)
	Ordinal       int       `nullable:"false" default:"0"`                // = Column(Integer, nullable=False, default=0)
	ImageId       string    `width:"36" charset:"ascii" nullable:"true"`  // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Name          string    `width:"64" charset:"ascii" nullable:"true"`  // Column(VARCHAR(64, charset='ascii'), nullable=True)
	Path          string    `width:"256" charset:"ascii" nullable:"true"` // Column(VARCHAR(256, charset='ascii'), nullable=True)
	Size          int64     `nullable:"false" default:"0"`                // = Column(Integer, nullable=False, default=0)
	UpdatedAt     time.Time `nullable:"false" updated_at:"true" nullable:"false"`
	UpdateVersion int       `default:"0" nullable:"false" auto_version:"true"`
}

func (self *SGuestfloppy) insertVfd(imageId string) bool {
	if len(self.ImageId) == 0 {
		_, err := db.Update(self, func() error {
			self.ImageId = imageId
			self.Name = ""
			self.Path = ""
			self.Size = 0
			return nil
		})
		if err != nil {
			log.Errorf("insertVfd saveupdate fail: %s", err)
			return false
		}
		return true
	} else {
		return false
	}
}

func (self *SGuestfloppy) insertVfdSucc(imageId string, path string, size int64, name string) bool {
	if self.ImageId == imageId {
		_, err := db.Update(self, func() error {
			self.Name = name
			self.Path = path
			self.Size = size
			return nil
		})
		if err != nil {
			log.Errorf("insertIsoSucc saveUpdate fail %s", err)
			return false
		}
		return true
	} else {
		return false
	}
}

func (self *SGuestfloppy) ejectVfd() bool {
	if len(self.ImageId) > 0 {
		_, err := db.Update(self, func() error {
			self.ImageId = ""
			self.Name = ""
			self.Path = ""
			self.Size = 0
			return nil
		})
		if err != nil {
			log.Errorf("ejectVfd saveUpdate fail %s", err)
			return false
		}
		return true
	} else {
		return false
	}
}

func (self *SGuestfloppy) GetDetails() string {
	if len(self.ImageId) > 0 {
		if self.Size > 0 {
			return fmt.Sprintf("%s(%s/%dMB)", self.Name, self.ImageId, self.Size)
		} else {
			return fmt.Sprintf("%s(inserting)", self.ImageId)
		}
	} else {
		return ""
	}
}

func (self *SGuestfloppy) getJsonDesc() *api.GuestfloppyJsonDesc {
	if len(self.ImageId) > 0 && len(self.Path) > 0 {
		return &api.GuestfloppyJsonDesc{
			Ordinal: self.Ordinal,
			ImageId: self.ImageId,
			Path:    self.Path,
			Name:    self.Name,
			Size:    self.Size,
		}
	}
	return nil
}
