package google

import (
	"fmt"
	"strconv"
	"time"

	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"yunion.io/x/jsonutils"
)

func (region *SRegion) GetMonitorData(id, serverName, metricName string, since time.Time,
	until time.Time) (*jsonutils.JSONArray, error) {
	params := map[string]string{
		"filter": `metric.type="` + metricName + `" AND metric.labels.instance_name="` + serverName + `"`,
		//"filter":             "metric.type=" + metricName + " AND metric.labels.instance_name=" + serverName,
		"interval.startTime": since.Format(time.RFC3339),
		"interval.endTime":   until.Format(time.RFC3339),
		"view":               strconv.FormatInt(int64(monitoringpb.ListTimeSeriesRequest_FULL), 10),
	}
	resource := fmt.Sprintf("%s/%s/%s", "projects", id, "timeSeries")

	timeSeries, err := region.client.monitorListAll(resource, params)
	if err != nil {
		return nil, err
	}
	return timeSeries, nil
}
