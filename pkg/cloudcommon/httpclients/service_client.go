package httpclients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	DEFAULT_VERSION = "v1"
	MAX_TRY_TIMES   = 3
)

type SServiceClient struct {
	client                   *http.Client
	region, service, version string
}

func NewServiceClient(region, service, version string) *SServiceClient {
	return &SServiceClient{
		client:  httputils.GetDefaultClient(),
		region:  region,
		service: service,
		version: version,
	}
}

func (c *SServiceClient) Request(ctx context.Context, method string, urlStr string, header http.Header, body jsonutils.JSONObject, debug bool) (http.Header, jsonutils.JSONObject, error) {
	if _, ok := header["X-Auth-Token"]; !ok {
		token := auth.GetTokenString()
		if len(token) == 0 {
			return nil, nil, fmt.Errorf("Missing Auth Token")
		}
		header.Set("X-Auth-Token", token)
	}
	baseUrl, err := c.GetUrl()
	if err != nil {
		return nil, nil, err
	}
	urlStr = baseUrl + urlStr
	return httputils.JSONRequest(c.client, ctx, method, urlStr, header, body, debug)
}

func (c *SServiceClient) GetUrl() (string, error) {
	var service = c.service
	if c.version != DEFAULT_VERSION {
		service = fmt.Sprintf("%s_%s", service, c.version)
	}
	return auth.GetServiceURL(service, c.region, "", "")
}

func (c *SServiceClient) TaskFail(ctx context.Context, taskId string, reason jsonutils.JSONObject) {
	body := jsonutils.NewDict()
	body.Set("__status__", jsonutils.NewString("error"))
	body.Set("__reason__", reason)
	c.TaskComplete(ctx, taskId, body, 0)
}

func (c *SServiceClient) TaskComplete(ctx context.Context, taskId string, data jsonutils.JSONObject, tried int) {
	url := fmt.Sprintf("/tasks/%s", taskId)
	_, res, err := c.Request(ctx, "POST", url, nil, data, false)
	if err != nil {
		log.Errorf("Sync task complete fail %s", err)
		if tried < MAX_TRY_TIMES {
			time.Sleep(time.Second * 5)
			c.TaskComplete(ctx, taskId, data, tried+1)
		}
	} else {
		var output string
		if res != nil {
			output = res.String()
		}
		log.Infof("Sync task complete succ %s", output)
	}
}
