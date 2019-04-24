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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SGuestcdromManager struct {
	db.SModelBaseManager
}

var GuestcdromManager *SGuestcdromManager

func init() {
	GuestcdromManager = &SGuestcdromManager{
		SModelBaseManager: db.NewModelBaseManager(
			SGuestcdrom{},
			"guestcdrom_tbl",
			"guestcdrom",
			"guestcdroms",
		),
	}
}

type SGuestcdrom struct {
	db.SModelBase

	Id            string    `width:"36" charset:"ascii" primary:"true"`   // = Column(VARCHAR(36, charset='ascii'), primary_key=True)
	ImageId       string    `width:"36" charset:"ascii" nullable:"true"`  // Column(VARCHAR(36, charset='ascii'), nullable=True)
	Name          string    `width:"64" charset:"ascii" nullable:"true"`  // Column(VARCHAR(64, charset='ascii'), nullable=True)
	Path          string    `width:"256" charset:"ascii" nullable:"true"` // Column(VARCHAR(256, charset='ascii'), nullable=True)
	Size          int       `nullable:"false" default:"0"`                // = Column(Integer, nullable=False, default=0)
	UpdatedAt     time.Time `nullable:"false" updated_at:"true" nullable:"false"`
	UpdateVersion int       `default:"0" nullable:"false" auto_version:"true"`
}

func (self *SGuestcdrom) insertIso(imageId string) bool {
	if len(self.ImageId) == 0 {
		_, err := db.Update(self, func() error {
			self.ImageId = imageId
			self.Name = ""
			self.Path = ""
			self.Size = 0
			return nil
		})
		if err != nil {
			log.Errorf("insertISO saveupdate fail: %s", err)
			return false
		}
		return true
	} else {
		return false
	}
}

func (self *SGuestcdrom) insertIsoSucc(imageId string, path string, size int, name string) bool {
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

func (self *SGuestcdrom) ejectIso() bool {
	if len(self.ImageId) > 0 {
		_, err := db.Update(self, func() error {
			self.ImageId = ""
			self.Name = ""
			self.Path = ""
			self.Size = 0
			return nil
		})
		if err != nil {
			log.Errorf("ejectIso saveUpdate fail %s", err)
			return false
		}
		return true
	} else {
		return false
	}
}

func (self *SGuestcdrom) GetDetails() string {
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

func (self *SGuestcdrom) getJsonDesc() jsonutils.JSONObject {
	if len(self.ImageId) > 0 && len(self.Path) > 0 {
		desc := jsonutils.NewDict()
		desc.Add(jsonutils.NewString(self.ImageId), "image_id")
		desc.Add(jsonutils.NewString(self.Path), "path")
		desc.Add(jsonutils.NewString(self.Name), "name")
		desc.Add(jsonutils.NewInt(int64(self.Size)), "size")
		return desc
	}
	return nil
}
