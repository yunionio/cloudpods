package models

import (
	"context"
	"time"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
)

type SVCenterManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	SInfrastructureManager
}

var VCenterManager *SVCenterManager

func init() {
	VCenterManager = &SVCenterManager{SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(SCloudprovider{}, "vcenters_tbl", "vcenter", "vcenters")}
}

type SVCenter struct {
	db.SEnabledStatusStandaloneResourceBase
	SInfrastructure

	Hostname string `width:"64" charset:"ascii" nullable:"false" list:"admin"` // = Column(VARCHAR(64, charset='ascii'), nullable=False)
	Port     int    `nullable:"false" list:"admin"`                            // = Column(Integer, nullable=False)
	Account  string `width:"64" charset:"ascii" nullable:"false" list:"admin"` // = Column(VARCHAR(64, charset='ascii'), nullable=False)
	Password string `width:"256" charset:"ascii" nullable:"false"`             // = Column(VARCHAR(256, charset='ascii'), nullable=False)

	LastSync time.Time `nullable:"true" get:"admin"` // = Column(DateTime, nullable=True)

	Version string `width:"32" charset:"ascii" nullable:"true" list:"admin"` // = Column(VARCHAR(32, charset='ascii'), nullable=True)

	Sysinfo jsonutils.JSONObject `nullable:"true" get:"admin"` // = Column(JSONEncodedDict, nullable=True)
}

func (manager *SVCenterManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}
