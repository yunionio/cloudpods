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

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/internal/models"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

type CloudpodsServerMonitorTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsServerMonitorTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsServerMonitorTool {
	return &CloudpodsServerMonitorTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsServerMonitorTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_get_server_monitor",
		mcp.WithDescription("获取Cloudpods虚拟机监控信息，包括CPU、内存、磁盘、网络等指标"),
		mcp.WithString("server_id",
			mcp.Required(),
			mcp.Description("虚拟机ID")),
		mcp.WithString("start_time",
			mcp.Description("开始时间戳（秒），默认为1小时前")),
		mcp.WithString("end_time",
			mcp.Description("结束时间戳（秒），默认为当前时间")),
		mcp.WithString("metrics",
			mcp.Description("监控指标，多个用逗号分隔，例如：cpu_usage,mem_usage,disk_usage,net_bps_rx,net_bps_tx")),
	)
}

func (c *CloudpodsServerMonitorTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	endTime := now
	startTime := now - 3600

	if startTimeStr := req.GetString("start_time", ""); startTimeStr != "" {
		if parsedStartTime, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil {
			startTime = parsedStartTime
		}
	}

	if endTimeStr := req.GetString("end_time", ""); endTimeStr != "" {
		if parsedEndTime, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil {
			endTime = parsedEndTime
		}
	}

	var metrics []string
	if metricsStr := req.GetString("metrics", ""); metricsStr != "" {
		metrics = strings.Split(metricsStr, ",")
		for i, metric := range metrics {
			metrics[i] = strings.TrimSpace(metric)
		}
	} else {
		metrics = []string{"cpu_usage", "mem_usage", "disk_usage", "net_bps_rx", "net_bps_tx"}
	}

	c.logger.WithFields(logrus.Fields{
		"server_id":  serverID,
		"start_time": startTime,
		"end_time":   endTime,
		"metrics":    metrics,
	}).Info("开始获取虚拟机监控信息")

	monitorResponse, err := c.adapter.GetServerMonitor(ctx, serverID, startTime, endTime, metrics)
	if err != nil {
		c.logger.WithError(err).Error("获取监控信息失败")
		return nil, fmt.Errorf("获取监控信息失败: %w", err)
	}

	formattedResult := c.formatMonitorResult(monitorResponse, serverID, startTime, endTime, metrics)

	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsServerMonitorTool) GetName() string {
	return "cloudpods_get_server_monitor"
}

func (c *CloudpodsServerMonitorTool) formatMonitorResult(response *models.MonitorResponse, serverID string, startTime, endTime int64, requestedMetrics []string) map[string]interface{} {
	formatted := map[string]interface{}{
		"query_info": map[string]interface{}{
			"server_id":         serverID,
			"start_time":        startTime,
			"end_time":          endTime,
			"start_time_human":  time.Unix(startTime, 0).Format("2006-01-02 15:04:05"),
			"end_time_human":    time.Unix(endTime, 0).Format("2006-01-02 15:04:05"),
			"requested_metrics": requestedMetrics,
			"duration_seconds":  endTime - startTime,
		},
		"status":  response.Status,
		"metrics": make([]map[string]interface{}, 0, len(response.Data.Metrics)),
	}

	for _, metric := range response.Data.Metrics {
		metricInfo := map[string]interface{}{
			"metric":      metric.Metric,
			"unit":        metric.Unit,
			"data_points": len(metric.Values),
			"values":      make([]map[string]interface{}, 0, len(metric.Values)),
		}

		var totalValue float64
		var minValue, maxValue float64
		var latestValue float64
		var latestTime int64

		for i, value := range metric.Values {
			valueInfo := map[string]interface{}{
				"timestamp":  value.Timestamp,
				"time_human": time.Unix(value.Timestamp, 0).Format("2006-01-02 15:04:05"),
				"value":      value.Value,
			}
			metricInfo["values"] = append(metricInfo["values"].([]map[string]interface{}), valueInfo)

			totalValue += value.Value
			if i == 0 {
				minValue = value.Value
				maxValue = value.Value
			} else {
				if value.Value < minValue {
					minValue = value.Value
				}
				if value.Value > maxValue {
					maxValue = value.Value
				}
			}

			if value.Timestamp > latestTime {
				latestTime = value.Timestamp
				latestValue = value.Value
			}
		}

		if len(metric.Values) > 0 {
			metricInfo["statistics"] = map[string]interface{}{
				"min":     minValue,
				"max":     maxValue,
				"average": totalValue / float64(len(metric.Values)),
				"latest":  latestValue,
			}
		}

		formatted["metrics"] = append(formatted["metrics"].([]map[string]interface{}), metricInfo)
	}

	formatted["summary"] = map[string]interface{}{
		"total_metrics":    len(response.Data.Metrics),
		"query_successful": response.Status == 200,
		"time_range_hours": float64(endTime-startTime) / 3600,
	}

	return formatted
}

type CloudpodsServerStatsTool struct {
	adapter *adapters.CloudpodsAdapter
	logger  *logrus.Logger
}

func NewCloudpodsServerStatsTool(adapter *adapters.CloudpodsAdapter, logger *logrus.Logger) *CloudpodsServerStatsTool {
	return &CloudpodsServerStatsTool{
		adapter: adapter,
		logger:  logger,
	}
}

func (c *CloudpodsServerStatsTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_get_server_stats",
		mcp.WithDescription("获取Cloudpods虚拟机实时统计信息，包括CPU使用率、内存使用率、磁盘使用率和网络流量"),
		mcp.WithString("server_id",
			mcp.Required(),
			mcp.Description("虚拟机ID")),
	)
}

func (c *CloudpodsServerStatsTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	c.logger.WithField("server_id", serverID).Info("开始获取虚拟机统计信息")

	statsResponse, err := c.adapter.GetServerStats(ctx, serverID)
	if err != nil {
		c.logger.WithError(err).Error("获取统计信息失败")
		return nil, fmt.Errorf("获取统计信息失败: %w", err)
	}

	formattedResult := c.formatStatsResult(statsResponse, serverID)

	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		c.logger.WithError(err).Error("序列化结果失败")
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

func (c *CloudpodsServerStatsTool) GetName() string {
	return "cloudpods_get_server_stats"
}

func (c *CloudpodsServerStatsTool) formatStatsResult(response *models.ServerStatsResponse, serverID string) map[string]interface{} {
	formatted := map[string]interface{}{
		"server_id": serverID,
		"status":    response.Status,
		"stats": map[string]interface{}{
			"cpu_usage":    fmt.Sprintf("%.1f%%", response.Data.CPUUsage),
			"memory_usage": fmt.Sprintf("%.1f%%", response.Data.MemUsage),
			"disk_usage":   fmt.Sprintf("%.1f%%", response.Data.DiskUsage),
			"network": map[string]interface{}{
				"receive_bps":   response.Data.NetBpsRx,
				"transmit_bps":  response.Data.NetBpsTx,
				"receive_mbps":  fmt.Sprintf("%.2f Mbps", float64(response.Data.NetBpsRx)/(1024*1024)),
				"transmit_mbps": fmt.Sprintf("%.2f Mbps", float64(response.Data.NetBpsTx)/(1024*1024)),
			},
			"updated_at": response.Data.UpdatedAt,
		},
		"raw_data": map[string]interface{}{
			"cpu_usage":  response.Data.CPUUsage,
			"mem_usage":  response.Data.MemUsage,
			"disk_usage": response.Data.DiskUsage,
			"net_bps_rx": response.Data.NetBpsRx,
			"net_bps_tx": response.Data.NetBpsTx,
		},
	}

	var healthStatus string
	var healthScore int

	if response.Data.CPUUsage > 90 || response.Data.MemUsage > 90 || response.Data.DiskUsage > 90 {
		healthStatus = "警告"
		healthScore = 1
	} else if response.Data.CPUUsage > 70 || response.Data.MemUsage > 70 || response.Data.DiskUsage > 80 {
		healthStatus = "注意"
		healthScore = 2
	} else {
		healthStatus = "正常"
		healthScore = 3
	}

	formatted["health"] = map[string]interface{}{
		"status": healthStatus,
		"score":  healthScore,
		"notes":  []string{},
	}

	notes := []string{}
	if response.Data.CPUUsage > 90 {
		notes = append(notes, "CPU使用率过高，建议检查系统负载")
	}
	if response.Data.MemUsage > 90 {
		notes = append(notes, "内存使用率过高，建议释放内存或增加内存")
	}
	if response.Data.DiskUsage > 90 {
		notes = append(notes, "磁盘使用率过高，建议清理磁盘空间")
	}
	formatted["health"].(map[string]interface{})["notes"] = notes

	return formatted
}
