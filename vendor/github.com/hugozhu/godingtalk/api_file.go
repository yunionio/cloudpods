package godingtalk

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
)

/**
 * https://open-doc.dingtalk.com/docs/doc.htm?spm=a219a.7629140.0.0.UeYQVr&treeId=172&articleId=104970&docType=1
 * TODO: not completed yet
 **/

//FileResponse is
type FileResponse struct {
	OAPIResponse
	Code     int
	Msg      string
	UploadID string `json:"uploadid"`
	Writer   io.Writer
}

func (f *FileResponse) getWriter() io.Writer {
	return f.Writer
}

//CreateFile is to create a new file in Ding Space
func (c *DingTalkClient) CreateFile(size int64) (file FileResponse, err error) {
	buf := bytes.Buffer{}
	file = FileResponse{
		Writer: &buf,
	}
	params := url.Values{}
	params.Add("size", fmt.Sprintf("%d", size))
	err = c.httpRPC("file/upload/create", params, nil, &file)
	return file, err
}
