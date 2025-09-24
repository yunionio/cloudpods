package s3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/internal/signer/open_api"
)

type QueryKs3DataInput struct {
	// 查询用量开始时间，格式为：yyyyMMdd，如：20250901 表示查询从2025年9月1日0点0分开始的用量
	StartTime *string `location:"querystring" locationName:"StartTime" type:"string" required:"true"`

	// 查询用量结束时间，格式为：yyyyMMdd，如：20250902 表示查询到2025年9月2日23点59分结束的用量
	EndTime *string `location:"querystring" locationName:"EndTime" type:"string" required:"true"`

	// 支持按天粒度查询，固定值：Day
	DateType *string `location:"querystring" locationName:"DateType" type:"string"`

	// 存储空间名称，最多支持同时查询5个存储桶的用量明细
	BucketNames []string `location:"querystrings" locationName:"Bucketname" type:"list" required:"true"`

	// 可以查询单个或多个计费项，如果不填，则查询除带宽外的所有计费项
	// DataSize：存储量
	// NetworkFlowUp：外网上行流量
	// NetworkFlow：外网下行流量
	// CDNFlow：CDN回源流量
	// ReplicationFlow：跨区域复制流量
	// RequestsGet：GET类请求次数
	// RequestsPut：PUT类请求次数
	// RestoreSize：数据取回量
	// TagNum：对象标签梳理
	// BandwidthUp：上行带宽（不区分外网和CDN）
	// BandwidthDown：下行带宽（不区分外网和CDN）
	// NetBandwidthUp：外网上行带宽
	// NetBandwidthDown：外网下行带宽
	// CDNBandwidthDown：CDN回源带宽
	// IntranetBandwidthUp：内网上行带宽
	// IntranetBandwidthDown：内网下行带宽
	// IntranetFlowUp：内网上行流量
	// IntranetFlowDown：内网下行流量
	// ObjectNum：桶内的对象数量
	Ks3Products []string `location:"querystrings" locationName:"Ks3Product" type:"list"`

	// 可以查询单个或多个统计项的流量情况，可选值：Object、Referer、IP、UA，返回TOP200数据
	Transfers []string `location:"querystrings" locationName:"Transfer" type:"list"`

	// 可以查询单个或多个统计项的请求次数情况，可选值：Object、Referer、IP、UA，返回TOP200数据
	Requests []string `location:"querystrings" locationName:"Request" type:"list"`
}

type QueryKs3DataOutput struct {
	// 查询结果
	Ks3DataResult *Ks3DataResult `locationName:"Ks3DataResult" type:"structure"`

	// 响应头
	Metadata map[string]*string `location:"headers" type:"map"`

	// HTTP 状态码
	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataQueryKs3DataOutput `json:"-" xml:"-"`
}

type metadataQueryKs3DataOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"Ks3DataResult"`
}

type Ks3DataResult struct {
	// 响应码
	Code *string `locationName:"Code" type:"string"`

	// 响应信息
	Message *string `locationName:"Message" type:"string"`

	// 包含一个或多个Bucket的容器
	Data *Ks3Data `locationName:"Data" type:"structure"`

	// 请求ID
	RequestId *string `locationName:"RequestId" type:"string"`
}

type Ks3Data struct {
	// 包含一个或多个Bucket的列表
	Buckets []*Ks3DataBucket `locationName:"Buckets" type:"list"`
}

type Ks3DataBucket struct {
	// Bucket的名称
	Name *string `locationName:"Name" type:"string"`

	// 数据开始时间
	StartTime *string `locationName:"StartTime" type:"string"`

	// 数据结束时间
	EndTime *string `locationName:"EndTime" type:"string"`

	// Bucket的标准存储量，单位是Bytes
	StandardDataSize *string `locationName:"StandardDataSize" type:"string"`

	// Bucket的低频存储量，单位是Bytes
	StandardIADataSize *string `locationName:"StandardIADataSize" type:"string"`

	// Bucket的归档存储量，单位是Bytes
	ArchiveDataSize *string `locationName:"ArchiveDataSize" type:"string"`

	// 标准存储的PUT请求次数
	StandardPutRequest *string `locationName:"StandardPutRequest" type:"string"`

	// 低频存储的PUT请求次数
	StandardIAPutRequest *string `locationName:"StandardIAPutRequest" type:"string"`

	// 归档存储的PUT请求次数
	ArchivePutRequest *string `locationName:"ArchivePutRequest" type:"string"`

	// 标准存储的GET请求次数
	StandardGetRequest *string `locationName:"StandardGetRequest" type:"string"`

	// 低频存储的GET请求次数
	StandardIAGetRequest *string `locationName:"StandardIAGetRequest" type:"string"`

	// 归档存储的GET请求次数
	ArchiveGetRequest *string `locationName:"ArchiveGetRequest" type:"string"`

	// 外网下行流量，单位是Bytes
	NetworkFlow *string `locationName:"NetworkFlow" type:"string"`

	// CDN回源流量，单位是Bytes
	CDNFlow *string `locationName:"CDNFlow" type:"string"`

	// 跨区域复制流量，单位是Bytes
	ReplicationFlow *string `locationName:"ReplicationFlow" type:"string"`

	// 外网上行带宽，不区分外网上行和CDN上行，单位是bps
	BandwidthUp []map[string]*string `locationName:"BandwidthUp" type:"list"`

	// 外网下行带宽，不区分外网下行和CDN下行，单位是bps
	BandwidthDown []map[string]*string `locationName:"BandwidthDown" type:"list"`

	// 外网下行带宽，单位是bps
	OuterBandwidthDown []map[string]*string `locationName:"OuterBandwidthDown" type:"list"`

	// CDN回源带宽，单位是bps
	CDNBandwidthDown []map[string]*string `locationName:"CDNBandwidthDown" type:"list"`

	// 低频存储数据取回量，单位是Bytes
	StandardIAData *string `locationName:"StandardIAData" type:"string"`

	// 外网上行带宽
	NetBandwidthUp []map[string]*string `locationName:"NetBandwidthUp" type:"list"`

	// 外网上行流量
	NetworkFlowUp *string `locationName:"NetworkFlowUp" type:"string"`

	// 内网上行带宽
	IntranetBandwidthUp []map[string]*string `locationName:"IntranetBandwidthUp" type:"list"`

	// 内网上行流量
	IntranetFlowUp *string `locationName:"IntranetFlowUp" type:"string"`

	// 内网下行带宽
	IntranetBandwidthDown []map[string]*string `locationName:"IntranetBandwidthDown" type:"list"`

	// 内网下行流量
	IntranetFlowDown *string `locationName:"IntranetFlowDown" type:"string"`

	// 桶内的对象数量
	ObjectNum *string `locationName:"ObjectNum" type:"string"`

	// 归档存储解冻数据量，单位是Bytes
	ArchiveData *string `locationName:"ArchiveData" type:"string"`

	// 对象标签的数量
	TagNum *string `locationName:"TagNum" type:"string"`

	// Object、Referer、IP、UA产生的流量
	Transfer *Ks3DataTransfer `locationName:"Transfer" type:"structure"`

	// Object、Referer、IP、UA产生的请求次数
	Request *Ks3DataRequest `locationName:"Request" type:"structure"`
}

type Ks3DataTransfer struct {
	// 指定Object产生的流量
	Objects []*Ks3DataTransferObject `locationName:"Object" type:"list"`

	// 指定Referer产生的流量
	Referers []*Ks3DataTransferReferer `locationName:"Referer" type:"list"`

	// 指定IP产生的流量
	IPs []*Ks3DataTransferIP `locationName:"Ip" type:"list"`

	// 指定UA产生的流量
	UAs []*Ks3DataTransferUA `locationName:"Ua" type:"list"`
}

type Ks3DataTransferObject struct {
	// Object名称
	Object *string `locationName:"object" type:"string"`

	// Object产生的流量
	Traffic *string `locationName:"traffic" type:"string"`
}

type Ks3DataTransferReferer struct {
	// Referer名称
	Referer *string `locationName:"referer" type:"string"`

	// Referer产生的流量
	Traffic *string `locationName:"traffic" type:"string"`
}

type Ks3DataTransferIP struct {
	// IP地址
	IP *string `locationName:"ip" type:"string"`

	// IP产生的流量
	Traffic *string `locationName:"traffic" type:"string"`
}

type Ks3DataTransferUA struct {
	// UA名称
	UA *string `locationName:"ua" type:"string"`

	// UA产生的流量
	Traffic *string `locationName:"traffic" type:"string"`
}

type Ks3DataRequest struct {
	// 指定Object产生的流量
	Objects []*Ks3DataRequestObject `locationName:"Object" type:"list"`

	// 指定Referer产生的流量
	Referers []*Ks3DataRequestReferer `locationName:"Referer" type:"list"`

	// 指定IP产生的流量
	IPs []*Ks3DataRequestIP `locationName:"Ip" type:"list"`

	// 指定UA产生的流量
	UAs []*Ks3DataRequestUA `locationName:"Ua" type:"list"`
}

type Ks3DataRequestObject struct {
	// Object名称
	Object *string `locationName:"object" type:"string"`

	// Object产生的请求次数
	Times *string `locationName:"times" type:"string"`
}

type Ks3DataRequestReferer struct {
	// Referer名称
	Referer *string `locationName:"referer" type:"string"`

	// Referer产生的请求次数
	Times *string `locationName:"times" type:"string"`
}

type Ks3DataRequestIP struct {
	// IP地址
	IP *string `locationName:"ip" type:"string"`

	// IP产生的请求次数
	Times *string `locationName:"times" type:"string"`
}

type Ks3DataRequestUA struct {
	// UA名称
	UA *string `locationName:"ua" type:"string"`

	// UA产生的请求次数
	Times *string `locationName:"times" type:"string"`
}

// QueryKs3DataRequest generates a request for the QueryKs3DataRequest operation.
func (c *S3) QueryKs3DataRequest(input *QueryKs3DataInput) (req *aws.Request, output *QueryKs3DataOutput) {
	op := &aws.Operation{
		Name:       "QueryKs3Data",
		HTTPMethod: "GET",
		HTTPPath:   "/?Action=QueryKs3Data",
	}

	if input == nil {
		input = &QueryKs3DataInput{}
	}

	if input.StartTime != nil && aws.ToString(input.StartTime) != "" {
		input.StartTime = aws.String(aws.ToString(input.StartTime) + "0000")
	}

	if input.EndTime != nil && aws.ToString(input.EndTime) != "" {
		input.EndTime = aws.String(aws.ToString(input.EndTime) + "2359")
	}

	if input.DateType == nil || aws.ToString(input.DateType) == "" {
		input.DateType = aws.String("Day")
	}

	req = c.newRequest(op, input, output)
	req.RequestType = "ks3bill"
	req.ContentType = "application/json"
	req.HTTPRequest.URL.Host = c.Config.Ks3BillEndpoint
	req.Handlers.Sign.Clear()
	req.Handlers.Sign.PushBack(open_api.Sign)
	output = &QueryKs3DataOutput{
		Ks3DataResult: &Ks3DataResult{},
	}
	req.Data = output
	return
}

// QueryKs3Data 桶用量详情及业务分析查询
func (c *S3) QueryKs3Data(input *QueryKs3DataInput) (*QueryKs3DataOutput, error) {
	req, out := c.QueryKs3DataRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) QueryKs3DataWithContext(ctx aws.Context, input *QueryKs3DataInput) (*QueryKs3DataOutput, error) {
	req, out := c.QueryKs3DataRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type QueryBucketRankInput struct {
	// 查询用量开始时间，格式为：yyyyMMdd，如：20250901 表示查询从2025年9月1日0点0分开始的用量
	StartTime *string `location:"querystring" locationName:"StartTime" type:"string" required:"true"`

	// 查询用量结束时间，格式为：yyyyMMdd，如：20250902 表示查询到2025年9月2日23点59分结束的用量
	EndTime *string `location:"querystring" locationName:"EndTime" type:"string" required:"true"`

	// 支持按天粒度查询，固定值：Day
	DateType *string `location:"querystring" locationName:"DateType" type:"string"`

	// 可以查询单个或多个统计项，如果不填，则查询所有统计项
	// DataSize：存储量
	// Flow：外网下行流量
	// RequestsGet：GET类请求次数
	// RequestsPut：PUT类请求次数
	Ks3Products []string `location:"querystrings" locationName:"Ks3Product" type:"list"`

	// TOP排序的Bucket数量，取值范围为[1-500]，默认值为200
	Number *int64 `location:"querystring" locationName:"Number" type:"integer"`
}

type QueryBucketRankOutput struct {
	// 查询结果
	BucketRankResult *BucketRankResult `locationName:"BucketRankResult" type:"structure"`

	// 响应头
	Metadata map[string]*string `location:"headers" type:"map"`

	// HTTP 状态码
	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataQueryBucketRankOutput `json:"-" xml:"-"`
}

type metadataQueryBucketRankOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"BucketRankResult"`
}

type BucketRankResult struct {
	// 响应码
	Code *string `locationName:"Code" type:"string"`

	// 响应信息
	Message *string `locationName:"Message" type:"string"`

	// 桶用量排序数据
	Data *BucketRankData `locationName:"Data" type:"structure"`

	// 请求ID
	RequestId *string `locationName:"RequestId" type:"string"`
}

type BucketRankData struct {
	// Bucket的存储量，单位是Bytes
	DataSize []map[string]*string `locationName:"DataSize" type:"list"`

	// Bucket的外网下行流量，单位是Bytes
	Flow []map[string]*string `locationName:"Flow" type:"list"`

	// Bucket的GET类请求次数，单位是次
	RequestsGet []map[string]*string `locationName:"RequestsGet" type:"list"`

	// Bucket的PUT类请求次数，单位是次
	RequestsPut []map[string]*string `locationName:"RequestsPut" type:"list"`
}

// QueryBucketRankRequest generates a request for the QueryBucketRankRequest operation.
func (c *S3) QueryBucketRankRequest(input *QueryBucketRankInput) (req *aws.Request, output *QueryBucketRankOutput) {
	op := &aws.Operation{
		Name:       "QueryBucketRank",
		HTTPMethod: "GET",
		HTTPPath:   "/?Action=QueryBucketRank",
	}

	if input == nil {
		input = &QueryBucketRankInput{}
	}

	if input.StartTime != nil && aws.ToString(input.StartTime) != "" {
		input.StartTime = aws.String(aws.ToString(input.StartTime) + "0000")
	}

	if input.EndTime != nil && aws.ToString(input.EndTime) != "" {
		input.EndTime = aws.String(aws.ToString(input.EndTime) + "2359")
	}

	if input.DateType == nil || aws.ToString(input.DateType) == "" {
		input.DateType = aws.String("Day")
	}

	if input.Number == nil {
		input.Number = aws.Long(200)
	}

	req = c.newRequest(op, input, output)
	req.RequestType = "ks3bill"
	req.ContentType = "application/json"
	req.HTTPRequest.URL.Host = c.Config.Ks3BillEndpoint
	req.Handlers.Sign.Clear()
	req.Handlers.Sign.PushBack(open_api.Sign)
	output = &QueryBucketRankOutput{
		BucketRankResult: &BucketRankResult{},
	}
	req.Data = output
	return
}

// QueryBucketRank 桶用量排序查询
func (c *S3) QueryBucketRank(input *QueryBucketRankInput) (*QueryBucketRankOutput, error) {
	req, out := c.QueryBucketRankRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) QueryBucketRankWithContext(ctx aws.Context, input *QueryBucketRankInput) (*QueryBucketRankOutput, error) {
	req, out := c.QueryBucketRankRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}
