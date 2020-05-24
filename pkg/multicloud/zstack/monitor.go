package zstack

import (
	"time"

	"yunion.io/x/jsonutils"
)

type SDataPoint struct {
	DataPoints []DataPoint `json:"data"`
}

type DataPoint struct {
	Value     float64 `json:"value"`
	TimeStemp int64   `json:"time"`
	Labels    *Label  `json:"labels"`
}

type Label struct {
	VMUuid   string `json:"VMUuid"`
	HostUuid string `json:"HostUuid"`
}

func (region *SRegion) GetMonitorData(name string, namespace string, since time.Time,
	until time.Time) (*SDataPoint, error) {
	datas := SDataPoint{}
	param := jsonutils.NewDict()
	param.Add(jsonutils.NewString(namespace), "namespace")
	param.Add(jsonutils.NewString(name), "metricName")
	param.Add(jsonutils.NewString("60"), "period")
	param.Add(jsonutils.NewInt(since.Unix()), "startTime")
	param.Add(jsonutils.NewInt(until.Unix()), "endTime")
	rep, err := region.client.getMonitor("zwatch/metrics", param)
	if err != nil {
		return nil, err
	}
	rep.Unmarshal(&datas)
	return &datas, nil
}
