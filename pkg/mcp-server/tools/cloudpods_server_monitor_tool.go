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

	"github.com/mark3labs/mcp-go/mcp"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcp-server/adapters"
	"yunion.io/x/onecloud/pkg/mcp-server/models"
)

// CloudpodsServerMonitorTool 用于获取Cloudpods虚拟机监控信息
type CloudpodsServerMonitorTool struct {
	// adapter 用于与 Cloudpods API 进行交互
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServerMonitorTool 创建一个新的CloudpodsServerMonitorTool实例
// adapter: 用于与Cloudpods API交互的适配器
// 返回值: CloudpodsServerMonitorTool实例指针
func NewCloudpodsServerMonitorTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServerMonitorTool {
	return &CloudpodsServerMonitorTool{
		adapter: adapter,
	}
}

// GetTool 定义并返回获取虚拟机监控信息工具的元数据
// 该工具用于获取Cloudpods虚拟机的监控信息，包括CPU、内存、磁盘、网络等指标
// server_id: 虚拟机ID (必填)
// start_time: 开始时间戳（秒），默认为1小时前
// end_time: 结束时间戳（秒），默认为当前时间
// metrics: 监控指标，多个用逗号分隔，例如：cpu_usage,mem_usage,disk_usage,net_bps_rx,net_bps_tx
// ak: 用户登录cloudpods后获取的access key
// sk: 用户登录cloudpods后获取的secret key
func (c *CloudpodsServerMonitorTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_get_server_monitor",
		mcp.WithDescription("获取Cloudpods虚拟机监控信息，包括CPU、内存、磁盘、网络等指标"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("虚拟机ID")),
		mcp.WithString("start_time", mcp.Description("开始时间戳（秒），默认为1小时前")),
		mcp.WithString("end_time", mcp.Description("结束时间戳（秒），默认为当前时间")),
		mcp.WithString("metrics", mcp.Description("监控指标，多个用逗号分隔，例如：cpu_usage,mem_usage,disk_usage,net_bps_rx,net_bps_tx")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理获取虚拟机监控信息的请求
// ctx: 控制生命周期的上下文
// req: 包含获取监控信息所需参数的请求对象
// 返回值: 包含监控信息的响应对象和可能的错误
func (c *CloudpodsServerMonitorTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取必填参数：虚拟机ID
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	// 设置默认时间范围：结束时间为当前时间，开始时间为1小时前
	now := time.Now().Unix()
	startTime := now - 3600

	// 解析开始时间参数，如果指定则使用指定值
	if startTimeStr := req.GetString("start_time", ""); startTimeStr != "" {
		if parsedStartTime, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil {
			startTime = parsedStartTime
		}
	}

	// 解析结束时间参数，如果指定则使用指定值
	endTime := now
	if endTimeStr := req.GetString("end_time", ""); endTimeStr != "" {
		if parsedEndTime, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil {
			endTime = parsedEndTime
		}
	}

	// 获取可选参数：监控指标
	var metrics []string
	if metricsStr := req.GetString("metrics", ""); metricsStr != "" {
		metrics = strings.Split(metricsStr, ",")
		for i, metric := range metrics {
			metrics[i] = strings.TrimSpace(metric)
		}
	} else {
		metrics = []string{"cpu_usage", "mem_usage", "disk_usage", "net_bps_rx", "net_bps_tx"}
	}

	// 获取ak和sk参数，用于认证
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器获取虚拟机监控信息
	monitorResponse, err := c.adapter.GetServerMonitor(ctx, serverID, startTime, endTime, metrics, ak, sk)
	if err != nil {
		log.Errorf("Fail to get server monitor: %s", err)
		return nil, fmt.Errorf("fail to get server monitor: %w", err)
	}

	// 格式化监控结果
	formattedResult := c.formatMonitorResult(monitorResponse, serverID, startTime, endTime, metrics)

	// 将结果序列化为JSON格式
	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		log.Errorf("Fail to serialize result: %s", err)
		return nil, fmt.Errorf("fail to serialize result: %w", err)
	}

	// 返回格式化后的结果
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回工具的名称标识符
// 返回值: 工具名称字符串，用于唯一标识该工具
func (c *CloudpodsServerMonitorTool) GetName() string {
	return "cloudpods_get_server_monitor"
}

// formatMonitorResult 格式化虚拟机监控信息的响应结果
// response: 原始监控响应数据
// serverID: 虚拟机ID
// startTime: 监控开始时间
// endTime: 监控结束时间
// requestedMetrics: 请求的监控指标
// 返回值: 包含监控信息的格式化结果
func (c *CloudpodsServerMonitorTool) formatMonitorResult(response *models.MonitorResponse, serverID string, startTime, endTime int64, requestedMetrics []string) map[string]interface{} {
	// 初始化格式化结果结构
	formatted := map[string]interface{}{
		// 添加请求的基本信息
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

// CloudpodsServerStatsTool 用于获取Cloudpods虚拟机实时统计信息
type CloudpodsServerStatsTool struct {
	// adapter 用于与 Cloudpods API 进行交互
	adapter *adapters.CloudpodsAdapter
}

// NewCloudpodsServerStatsTool 创建一个新的CloudpodsServerStatsTool实例
// adapter: 用于与Cloudpods API交互的适配器
// 返回值: CloudpodsServerStatsTool实例指针
func NewCloudpodsServerStatsTool(adapter *adapters.CloudpodsAdapter) *CloudpodsServerStatsTool {
	return &CloudpodsServerStatsTool{
		adapter: adapter,
	}
}

// GetTool 定义并返回获取虚拟机统计信息工具的元数据
// 该工具用于获取Cloudpods虚拟机的实时统计信息，包括CPU使用率、内存使用率、磁盘使用率和网络流量
// server_id: 虚拟机ID (必填)
// ak: 用户登录cloudpods后获取的access key
// sk: 用户登录cloudpods后获取的secret key
func (c *CloudpodsServerStatsTool) GetTool() mcp.Tool {
	return mcp.NewTool(
		"cloudpods_get_server_stats",
		mcp.WithDescription("获取Cloudpods虚拟机实时统计信息，包括CPU使用率、内存使用率、磁盘使用率和网络流量"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("虚拟机ID")),
		mcp.WithString("ak", mcp.Description("用户登录cloudpods后获取的access key")),
		mcp.WithString("sk", mcp.Description("用户登录cloudpods后获取的secret key")),
	)
}

// Handle 处理获取虚拟机统计信息的请求
// ctx: 控制生命周期的上下文
// req: 包含获取统计信息所需参数的请求对象
// 返回值: 包含统计信息的响应对象和可能的错误
func (c *CloudpodsServerStatsTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取必填参数：虚拟机ID
	serverID, err := req.RequireString("server_id")
	if err != nil {
		return nil, err
	}

	// 获取可选参数：访问凭证
	ak := req.GetString("ak", "")
	sk := req.GetString("sk", "")

	// 调用适配器获取虚拟机统计信息
	statsResponse, err := c.adapter.GetServerStats(ctx, serverID, ak, sk)
	if err != nil {
		log.Errorf("Fail to get server stats: %s", err)
		return nil, fmt.Errorf("fail to get server stats: %w", err)
	}

	// 格式化统计结果
	formattedResult := c.formatStatsResult(statsResponse, serverID)

	// 将结果序列化为JSON格式
	resultJSON, err := json.MarshalIndent(formattedResult, "", "  ")
	if err != nil {
		log.Errorf("Fail to serialize result: %s", err)
		return nil, fmt.Errorf("fail to serialize result: %w", err)
	}

	// 返回格式化后的结果
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// GetName 返回工具的名称标识符
// 返回值: 工具名称字符串，用于唯一标识该工具
func (c *CloudpodsServerStatsTool) GetName() string {
	return "cloudpods_get_server_stats"
}

// formatStatsResult 格式化虚拟机统计信息的响应结果
// response: 原始统计响应数据
// serverID: 虚拟机ID
// 返回值: 包含统计信息的格式化结果
func (c *CloudpodsServerStatsTool) formatStatsResult(response *models.ServerStatsResponse, serverID string) map[string]interface{} {
	// 初始化格式化结果结构
	formatted := map[string]interface{}{
		"server_id": serverID,
		"status":    response.Status,
		// 添加统计信息
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
		// 添加原始数据
		"raw_data": map[string]interface{}{
			"cpu_usage":  response.Data.CPUUsage,
			"mem_usage":  response.Data.MemUsage,
			"disk_usage": response.Data.DiskUsage,
			"net_bps_rx": response.Data.NetBpsRx,
			"net_bps_tx": response.Data.NetBpsTx,
		},
	}

	// 评估虚拟机健康状态
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

	// 添加健康状态信息
	formatted["health"] = map[string]interface{}{
		"status": healthStatus,
		"score":  healthScore,
		"notes":  []string{},
	}

	// 添加健康状态建议
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
