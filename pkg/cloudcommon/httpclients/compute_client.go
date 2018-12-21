package httpclients

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/cloudcommon/httpclients"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

type SComputeClient struct {
	SServiceClient
}

func NewComputeClient(region, service, version string) *SComputeClient {
	return &SComputeClient{SServiceClient: *NewServiceClient(region, service, version)}
}

var computeClients map[string]*SComputeClient

func init() {
	computeClients = make(map[string]*SComputeClient, 0)
}

func GetComputeClient(region, version string) *SComputeClient {
	if len(version) == 0 {
		version = DEFAULT_VERSION
	}
	if len(region) == 0 {
		region = options.HostOptions.Region
	}
	if len(region) == 0 {
		return nil
	}
	cli, ok := computeClients[region+"-"+version]
	if !ok {
		log.Infof("Compute service client for region %s-%s initialized", region, version)
		cli = NewComputeClient(region, "compute", version)
		computeClients[region+"-"+version] = cli
		return cli
	}
	return cli
}

func GetDefaultComputeClient() *SComputeClient {
	if len(options.HostOptions.Region) == 0 {
		return nil
	}
	cli, ok := computeClients[options.HostOptions.Region+"-"+DEFAULT_VERSION]
	if !ok {
		log.Infof("Compute service client for region %s-%s initialized",
			options.HostOptions.Region, DEFAULT_VERSION)
		cli = NewComputeClient(options.HostOptions.Region, "compute", DEFAULT_VERSION)

	}
	return cli
}

func (c *SComputeClient) UpdateServerStatus(sid, status string) {
	var url = fmt.Sprintf("/servers/%s/status", sid)
	var body = jsonutils.NewDict()
	var stus = jsonutils.NewDict()
	stus.Set("status", jsonutils.NewString(status))
	body.Set("server", stus)
	c.Request(context.Background(), "POST", url, nil, body, false)
}

func TaskFailed(ctx context.Context, reason string) error {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		httpclients.GetDefaultComputeClient().TaskFail(ctx, taskId.(string), reason)
		return nil
	} else {
		log.Errorln("Reqeuest task failed missing task id, with reason(%s)", reason)
		return fmt.Errorf("Reqeuest task failed missing task id")
	}
}

func TaskComplete(ctx context.Context, data jsonutils.JSONObject) error {
	if taskId := ctx.Value(appctx.APP_CONTEXT_KEY_TASK_ID); taskId != nil {
		httpclients.GetDefaultComputeClient().TaskComplete(ctx, taskId.(string), data, 0)
		return nil
	} else {
		log.Errorln("Reqeuest task complete missing task id")
		return fmt.Errorf("Reqeuest task complete missing task id")
	}
}
