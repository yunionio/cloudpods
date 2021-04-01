package cos

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"hash/crc64"
	"io"
	"net/http"
	"os"
)

type CIService service

type PicOperations struct {
	IsPicInfo int                  `json:"is_pic_info,omitempty"`
	Rules     []PicOperationsRules `json:"rules,omitemtpy"`
}
type PicOperationsRules struct {
	Bucket string `json:"bucket,omitempty"`
	FileId string `json:"fileid"`
	Rule   string `json:"rule"`
}

func EncodePicOperations(pic *PicOperations) string {
	if pic == nil {
		return ""
	}
	bs, err := json.Marshal(pic)
	if err != nil {
		return ""
	}
	return string(bs)
}

type ImageProcessResult struct {
	XMLName        xml.Name          `xml:"UploadResult"`
	OriginalInfo   *PicOriginalInfo  `xml:"OriginalInfo,omitempty"`
	ProcessResults *PicProcessObject `xml:"ProcessResults>Object,omitempty"`
}
type PicOriginalInfo struct {
	Key       string        `xml:"Key,omitempty"`
	Location  string        `xml:"Location,omitempty"`
	ImageInfo *PicImageInfo `xml:"ImageInfo,omitempty"`
	ETag      string        `xml:"ETag,omitempty"`
}
type PicImageInfo struct {
	Format      string `xml:"Format,omitempty"`
	Width       int    `xml:"Width,omitempty"`
	Height      int    `xml:"Height,omitempty"`
	Quality     int    `xml:"Quality,omitempty"`
	Ave         string `xml:"Ave,omitempty"`
	Orientation int    `xml:"Orientation,omitempty"`
}
type PicProcessObject struct {
	Key             string `xml:"Key,omitempty"`
	Location        string `xml:"Location,omitempty"`
	Format          string `xml:"Format,omitempty"`
	Width           int    `xml:"Width,omitempty"`
	Height          int    `xml:"Height,omitempty"`
	Size            int    `xml:"Size,omitempty"`
	Quality         int    `xml:"Quality,omitempty"`
	ETag            string `xml:"ETag,omitempty"`
	WatermarkStatus int    `xml:"WatermarkStatus,omitempty"`
}

type picOperationsHeader struct {
	PicOperations string `header:"Pic-Operations" xml:"-" url:"-"`
}

type ImageProcessOptions = PicOperations

// 云上数据处理 https://cloud.tencent.com/document/product/460/18147
func (s *CIService) ImageProcess(ctx context.Context, name string, opt *ImageProcessOptions) (*ImageProcessResult, *Response, error) {
	header := &picOperationsHeader{
		PicOperations: EncodePicOperations(opt),
	}
	var res ImageProcessResult
	sendOpt := sendOptions{
		baseURL:   s.client.BaseURL.BucketURL,
		uri:       "/" + encodeURIComponent(name) + "?image_process",
		method:    http.MethodPost,
		optHeader: header,
		result:    &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}

type ImageRecognitionOptions struct {
	CIProcess  string `url:"ci-process,omitempty"`
	DetectType string `url:"detect-type,omitempty"`
}

type ImageRecognitionResult struct {
	XMLName       xml.Name         `xml:"RecognitionResult"`
	PornInfo      *RecognitionInfo `xml:"PornInfo,omitempty"`
	TerroristInfo *RecognitionInfo `xml:"TerroristInfo,omitempty"`
	PoliticsInfo  *RecognitionInfo `xml:"PoliticsInfo,omitempty"`
	AdsInfo       *RecognitionInfo `xml:"AdsInfo,omitempty"`
}
type RecognitionInfo struct {
	Code    int    `xml:"Code,omitempty"`
	Msg     string `xml:"Msg,omitempty"`
	HitFlag int    `xml:"HitFlag,omitempty"`
	Score   int    `xml:"Score,omitempty"`
	Label   string `xml:"Label,omitempty"`
	Count   int    `xml:"Count,omitempty"`
}

// 图片审核 https://cloud.tencent.com/document/product/460/37318
func (s *CIService) ImageRecognition(ctx context.Context, name string, opt *ImageRecognitionOptions) (*ImageRecognitionResult, *Response, error) {
	if opt != nil && opt.CIProcess == "" {
		opt.CIProcess = "sensitive-content-recognition"
	}
	var res ImageRecognitionResult
	sendOpt := sendOptions{
		baseURL:  s.client.BaseURL.BucketURL,
		uri:      "/" + encodeURIComponent(name),
		method:   http.MethodGet,
		optQuery: opt,
		result:   &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}

type PutVideoAuditingJobOptions struct {
	XMLName     xml.Name              `xml:"Request"`
	InputObject string                `xml:"Input>Object"`
	Conf        *VideoAuditingJobConf `xml:"Conf"`
}
type VideoAuditingJobConf struct {
	DetectType string                       `xml:",omitempty"`
	Snapshot   *PutVideoAuditingJobSnapshot `xml:",omitempty"`
	Callback   string                       `xml:",omitempty"`
}
type PutVideoAuditingJobSnapshot struct {
	Mode         string  `xml:",omitempty"`
	Count        int     `xml:",omitempty"`
	TimeInterval float32 `xml:",omitempty"`
	Start        float32 `xml:",omitempty"`
}

type PutVideoAuditingJobResult struct {
	XMLName    xml.Name `xml:"Response"`
	JobsDetail struct {
		JobId        string `xml:"JobId,omitempty"`
		State        string `xml:"State,omitempty"`
		CreationTime string `xml:"CreationTime,omitempty"`
		Object       string `xml:"Object,omitempty"`
	} `xml:"JobsDetail,omitempty"`
}

func (s *CIService) PutVideoAuditingJob(ctx context.Context, opt *PutVideoAuditingJobOptions) (*PutVideoAuditingJobResult, *Response, error) {
	var res PutVideoAuditingJobResult
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.CIURL,
		uri:     "/video/auditing",
		method:  http.MethodPost,
		body:    opt,
		result:  &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}

type GetVideoAuditingJobResult struct {
	XMLName        xml.Name                `xml:"Response"`
	JobsDetail     *VideoAuditingJobDetail `xml:",omitempty"`
	NonExistJobIds string                  `xml:",omitempty"`
}
type VideoAuditingJobDetail struct {
	Code          string                       `xml:",omitempty"`
	Message       string                       `xml:",omitempty"`
	JobId         string                       `xml:",omitempty"`
	State         string                       `xml:",omitempty"`
	CreationTime  string                       `xml:",omitempty"`
	Object        string                       `xml:",omitempty"`
	SnapshotCount string                       `xml:",omitempty"`
	Result        int                          `xml:",omitempty"`
	PornInfo      *RecognitionInfo             `xml:",omitempty"`
	TerrorismInfo *RecognitionInfo             `xml:",omitempty"`
	PoliticsInfo  *RecognitionInfo             `xml:",omitempty"`
	AdsInfo       *RecognitionInfo             `xml:",omitempty"`
	Snapshot      *GetVideoAuditingJobSnapshot `xml:",omitempty"`
}
type GetVideoAuditingJobSnapshot struct {
	Url           string           `xml:",omitempty"`
	PornInfo      *RecognitionInfo `xml:",omitempty"`
	TerrorismInfo *RecognitionInfo `xml:",omitempty"`
	PoliticsInfo  *RecognitionInfo `xml:",omitempty"`
	AdsInfo       *RecognitionInfo `xml:",omitempty"`
}

func (s *CIService) GetVideoAuditingJob(ctx context.Context, jobid string) (*GetVideoAuditingJobResult, *Response, error) {
	var res GetVideoAuditingJobResult
	sendOpt := sendOptions{
		baseURL: s.client.BaseURL.CIURL,
		uri:     "/video/auditing/" + jobid,
		method:  http.MethodGet,
		result:  &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)
	return &res, resp, err
}

// ci put https://cloud.tencent.com/document/product/460/18147
func (s *CIService) Put(ctx context.Context, name string, r io.Reader, uopt *ObjectPutOptions) (*ImageProcessResult, *Response, error) {
	if err := CheckReaderLen(r); err != nil {
		return nil, nil, err
	}
	opt := cloneObjectPutOptions(uopt)
	totalBytes, err := GetReaderLen(r)
	if err != nil && opt != nil && opt.Listener != nil {
		return nil, nil, err
	}
	if err == nil {
		// 与 go http 保持一致, 非bytes.Buffer/bytes.Reader/strings.Reader由用户指定ContentLength, 或使用 Chunk 上传
		if opt != nil && opt.ContentLength == 0 && IsLenReader(r) {
			opt.ContentLength = totalBytes
		}
	}
	reader := TeeReader(r, nil, totalBytes, nil)
	if s.client.Conf.EnableCRC {
		reader.writer = crc64.New(crc64.MakeTable(crc64.ECMA))
	}
	if opt != nil && opt.Listener != nil {
		reader.listener = opt.Listener
	}

	var res ImageProcessResult
	sendOpt := sendOptions{
		baseURL:   s.client.BaseURL.BucketURL,
		uri:       "/" + encodeURIComponent(name),
		method:    http.MethodPut,
		body:      reader,
		optHeader: opt,
		result:    &res,
	}
	resp, err := s.client.send(ctx, &sendOpt)

	return &res, resp, err
}

// ci put object from local file
func (s *CIService) PutFromFile(ctx context.Context, name string, filePath string, opt *ObjectPutOptions) (*ImageProcessResult, *Response, error) {
	fd, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer fd.Close()

	return s.Put(ctx, name, fd, opt)
}
