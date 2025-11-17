package models

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SGuestScreenDumpManager struct {
	db.SResourceBaseManager
}

var GuestScreenDumpManager *SGuestScreenDumpManager

func init() {
	db.InitManager(func() {
		GuestScreenDumpManager = &SGuestScreenDumpManager{
			SResourceBaseManager: db.NewResourceBaseManager(
				SGuestScreenDump{},
				"guest_screen_dumps_tbl",
				"guest_screen_dump",
				"guest_screen_dumps",
			),
		}
		GuestScreenDumpManager.SetVirtualObject(GuestScreenDumpManager)
		GuestScreenDumpManager.TableSpec().AddIndex(true, "guest_id")
	})
}

type SGuestScreenDump struct {
	db.SResourceBase

	RowId   int64  `primary:"true" auto_increment:"true" list:"user"`
	GuestId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	Name    string `width:"64" charset:"ascii" nullable:"true" list:"user"`

	// s3 config
	S3AccessKey  string `width:"64" charset:"ascii" nullable:"true"`
	S3SecretKey  string `width:"64" charset:"ascii" nullable:"true"`
	S3Endpoint   string `width:"64" charset:"ascii" nullable:"true" list:"user"`
	S3BucketName string `width:"64" charset:"ascii" nullable:"true" list:"user"`
	S3UseSsl     bool   `default:"false" list:"user" create:"optional"`
}

func (manager *SGuestScreenDumpManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestScreenDumpListInput,
) (*sqlchemy.SQuery, error) {
	if query.Server != "" {
		iGuest, err := GuestManager.FetchByIdOrName(ctx, userCred, query.Server)
		if err != nil {
			return q, errors.Wrap(err, "fetch guest")
		}
		q = q.Equals("guest_id", iGuest.GetId())
	}
	return q, nil
}

func (self *SGuestScreenDump) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGuest) SaveGuestScreenDump(ctx context.Context, userCred mcclient.TokenCredential, screenDumpInfo *api.SGuestScreenDump) (*SGuestScreenDump, error) {
	sd := new(SGuestScreenDump)
	sd.SetModelManager(GuestScreenDumpManager, sd)
	sd.GuestId = self.GetId()
	sd.S3SecretKey = base64.StdEncoding.EncodeToString([]byte(screenDumpInfo.S3SecretKey))
	sd.S3Endpoint = screenDumpInfo.S3Endpoint
	sd.S3BucketName = screenDumpInfo.S3BucketName
	sd.S3AccessKey = base64.StdEncoding.EncodeToString([]byte(screenDumpInfo.S3AccessKey))
	sd.Name = screenDumpInfo.S3ObjectName

	lockman.LockClass(ctx, GuestScreenDumpManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, GuestScreenDumpManager, self.ProjectId)

	err := GuestScreenDumpManager.TableSpec().Insert(ctx, sd)
	if err != nil {
		return nil, errors.Wrap(err, "save guest screen_dump")
	}
	db.OpsLog.LogEvent(self, db.ACT_GUEST_SCREEN_DUMP, sd.Name, userCred)
	logclient.AddSimpleActionLog(self, logclient.ACT_GUEST_SCREEN_DUMP, sd.Name, userCred, true)
	return sd, nil
}

func (self *SGuest) GetDetailsScreenDumpShow(ctx context.Context, userCred mcclient.TokenCredential, input *api.GetDetailsGuestScreenDumpInput) (*api.GetDetailsGuestScreenDumpOutput, error) {
	if input.ObjectName == "" {
		return nil, httperrors.NewMissingParameterError("object_name")
	}
	q := GuestScreenDumpManager.Query()
	q = q.Equals("guest_id", self.Id)
	q = q.Equals("name", input.ObjectName)

	screenDump := new(SGuestScreenDump)
	err := q.First(screenDump)
	if err != nil {
		return nil, errors.Wrap(err, "query screenDump")
	}

	ak, _ := base64.StdEncoding.DecodeString(screenDump.S3AccessKey)
	sk, _ := base64.StdEncoding.DecodeString(screenDump.S3SecretKey)

	url := screenDump.S3Endpoint
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		prefix := "http://"
		if screenDump.S3UseSsl {
			prefix = "https://"
		}
		url = prefix + url
	}
	cfg := objectstore.NewObjectStoreClientConfig(url, string(ak), string(sk))
	s3Client, err := objectstore.NewObjectStoreClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "new minio client")
	}
	bucket, err := s3Client.GetIBucketByName(screenDump.S3BucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "get bucket %s", screenDump.S3BucketName)
	}
	irc, err := bucket.GetObject(ctx, screenDump.Name, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get object %s", screenDump.Name)
	}
	defer irc.Close()
	obj, err := io.ReadAll(irc)
	if err != nil {
		return nil, errors.Wrapf(err, "read object %s", screenDump.Name)
	}
	ret := new(api.GetDetailsGuestScreenDumpOutput)

	contentType := http.DetectContentType(obj)
	base64Encoded := base64.StdEncoding.EncodeToString(obj)
	ret.ScreenDump = fmt.Sprintf("data:%s;base64,%s", contentType, base64Encoded)
	ret.ScreenDump = base64.StdEncoding.EncodeToString(obj)
	ret.GuestId = self.Id
	ret.Name = screenDump.Name
	return ret, nil
}

func (self *SGuest) PerformScreenDump(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.PowerStates != api.VM_POWER_STATES_ON {
		return nil, httperrors.NewBadRequestError("can't use qga in vm status: %s", self.Status)
	}

	host, _ := self.GetHost()
	driver, err := self.GetDriver()
	if err != nil {
		return nil, err
	}
	return driver.RequestGuestScreenDump(ctx, userCred, nil, host, self)
}
