package suggestsysdrivers

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type OssSecAcl struct {
	*baseDriver
}

func NewOssSecAclDriver() models.ISuggestSysRuleDriver {
	return &OssSecAcl{
		baseDriver: newBaseDriver(monitor.OSS_SEC_ACL, monitor.OSS_SEC_ACL_MONITOR_RES_TYPE,
			monitor.OSS_SEC_ACL_DRIVER_ACTION, monitor.OSS_SEC_ACL_MONITOR_SUGGEST),
	}
}

var aclUnSecurity = []string{"public-read", "public-read-write"}
var aclSecurity = "private"

func (drv *OssSecAcl) ValidateSetting(input *monitor.SSuggestSysAlertSetting) error {
	return nil
}

func (drv *OssSecAcl) DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	doSuggestSysRule(ctx, userCred, isStart, drv)
}

func (drv *OssSecAcl) Run(rule *models.SSuggestSysRule, setting *monitor.SSuggestSysAlertSetting) {
	Run(drv, rule, setting)
}

func (o OssSecAcl) StartResolveTask(ctx context.Context, userCred mcclient.TokenCredential, suggestSysAlert *models.SSuggestSysAlert,
	params *jsonutils.JSONDict) error {
	suggestSysAlert.SetStatus(userCred, monitor.SUGGEST_ALERT_START_DELETE, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ResoleBucketAclTask", suggestSysAlert, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (drv *OssSecAcl) Resolve(data *models.SSuggestSysAlert) error {
	session := auth.GetAdminSession(context.Background(), "", "")
	param := jsonutils.NewDict()
	keyArr := jsonutils.NewArray()
	bucket, err := modules.Buckets.GetById(session, data.ResId, jsonutils.NewDict())
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("get bucket by %s err", data.ResId))
	}

	param.Add(jsonutils.NewString(aclSecurity), "acl")

	bucketName, _ := bucket.GetString("name")
	problems := make([]monitor.SuggestAlertProblem, 0)
	err = data.Problem.Unmarshal(problems)
	if err != nil {
		return errors.Wrap(err, "unmarshal problem err")
	}
	bucketNameContainsBool := false
	for _, problem := range problems {
		if problem.Type == bucketName {
			bucketNameContainsBool = true
			continue
		}
		keyArr.Add(jsonutils.NewString(problem.Type))
	}
	if bucketNameContainsBool {
		//modify bucket acl
		_, err = modules.Buckets.PerformAction(session, data.ResId, "acl", param)
		if err != nil {
			return err
		}
	}

	//modify object acl in the bucket
	if keyArr.Length() == 0 {
		return nil
	}
	param.Add(keyArr, "key")
	_, err = modules.Buckets.PerformAction(session, data.ResId, "acl", param)
	return err
}

func (drv *OssSecAcl) GetLatestAlerts(rule *models.SSuggestSysRule, setting *monitor.SSuggestSysAlertSetting) ([]jsonutils.JSONObject, error) {
	allArr := make([]jsonutils.JSONObject, 0)
	bucketArr, err := drv.getBucketsByAcl()
	if err != nil {
		return nil, err
	}
	if bucketArr != nil {
		allArr = append(allArr, bucketArr...)
	}
	bucketObjArr, err := drv.getBucketsByObjAcl()
	if err != nil {
		return nil, err
	}
	if bucketObjArr != nil {
		allArr = append(allArr, bucketObjArr...)
	}
	return allArr, nil
}

func (drv *OssSecAcl) getBucketsByAcl() ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("acl.in(public-read,public-read-write)"), "filter")
	buckets, err := ListAllResources(&modules.Buckets, query)
	if err != nil {
		return nil, err
	}
	bucketArr := make([]jsonutils.JSONObject, 0)
	for _, bucket := range buckets {
		acl, _ := bucket.GetString("acl")
		bucketName, _ := bucket.GetString("name")
		suggestSysAlert, err := getSuggestSysAlertFromJson(bucket, drv)
		if err != nil {
			return bucketArr, errors.Wrap(err, "OssSecAcl getSuggestSysAlertFromJson error")
		}
		suggestSysAlert.Amount = 0
		problems := []monitor.SuggestAlertProblem{
			monitor.SuggestAlertProblem{
				Type:        bucketName,
				Description: acl,
			},
		}
		suggestSysAlert.Name = GenerateName(suggestSysAlert.Name, string(drv.GetType()))
		suggestSysAlert.Problem = jsonutils.Marshal(&problems)
		bucketArr = append(bucketArr, jsonutils.Marshal(suggestSysAlert))
	}
	return bucketArr, nil
}

func (drv *OssSecAcl) getBucketsByObjAcl() ([]jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(fmt.Sprintf("acl.equals(%s)", aclSecurity)), "filter")
	buckets, err := ListAllResources(&modules.Buckets, query)
	if err != nil {
		return nil, err
	}
	bucketArr := make([]jsonutils.JSONObject, 0)
	for _, bucket := range buckets {
		id, _ := bucket.GetString("id")
		unSecBool, problems, err := drv.getObjectsByBucketId(id, "")
		if err != nil {
			log.Errorln(err)
			continue
		}
		if !(*unSecBool) {
			continue
		}
		suggestSysAlert, err := getSuggestSysAlertFromJson(bucket, drv)
		if err != nil {
			return bucketArr, errors.Wrap(err, "OssSecAcl getSuggestSysAlertFromJson error")
		}
		suggestSysAlert.Amount = 0

		suggestSysAlert.Name = GenerateName(suggestSysAlert.Name, string(drv.GetType()))
		suggestSysAlert.Problem = jsonutils.Marshal(&problems)
		bucketArr = append(bucketArr, jsonutils.Marshal(suggestSysAlert))
	}
	return bucketArr, nil
}

func (drv *OssSecAcl) getObjectsByBucketId(id, filePath string) (*bool, []monitor.SuggestAlertProblem, error) {
	aclUnSafeBool := false
	problems := make([]monitor.SuggestAlertProblem, 0)
	objects, err := getBucketObjects(id, filePath)
	if err != nil {
		return nil, nil, err
	}
	for _, object := range objects {
		acl, _ := object.GetString("acl")
		key, _ := object.GetString("key")
		if acl != aclSecurity {
			aclUnSafeBool = true
			problems = append(problems, monitor.SuggestAlertProblem{
				Type:        key,
				Description: acl,
			})
		}
		if strings.HasSuffix(key, "/") {
			buffer := new(bytes.Buffer)
			if len(filePath) != 0 {
				buffer.WriteString(filePath)
			}
			buffer.WriteString(key)
			unSecBool, prob_, err := drv.getObjectsByBucketId(id, buffer.String())
			if err != nil {
				return nil, nil, err
			}
			if *unSecBool {
				problems = append(problems, prob_...)
			}
		}
	}
	return &aclUnSafeBool, problems, nil
}

func getBucketObjects(id, filePath string) ([]jsonutils.JSONObject, error) {
	session := auth.GetAdminSession(context.Background(), "", "")
	query := jsonutils.NewDict()
	if len(filePath) != 0 {
		query.Add(jsonutils.NewString(filePath), "prefix")
	}
	query.Add(jsonutils.NewString("system"), "scope")
	query.Add(jsonutils.NewInt(0), "limit")
	var count int
	objs := make([]jsonutils.JSONObject, 0)
	for {
		query.Add(jsonutils.NewInt(int64(count)), "offset")
		result, err := modules.Buckets.GetSpecific(session, id, "objects", query)
		if err != nil {
			return nil, errors.Wrapf(err, "%s getSpecific %s resources with params %s error",
				modules.Buckets.KeyString(), "objects", query.String())
		}
		listResult := modulebase.ListResult{}
		err = result.Unmarshal(&listResult)
		if err != nil {
			return nil, errors.Wrap(err, "getObjectsByBucketId unmarshal ListResult error")
		}
		for _, data := range listResult.Data {
			objs = append(objs, data)
		}
		total := listResult.Total
		count = count + len(listResult.Data)
		if count >= total {
			break
		}
	}
	return objs, nil
}
