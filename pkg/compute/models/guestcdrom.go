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
