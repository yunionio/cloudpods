package godingtalk

import (
	"io"
	"net/url"
	"time"
)

//MediaResponse is
type MediaResponse struct {
	OAPIResponse
	Type    string
	MediaID string `json:"media_id"`
	Writer  io.Writer
}

func (m *MediaResponse) getWriter() io.Writer {
	return m.Writer
}

//UploadMedia is to upload media file to DingTalk
func (c *DingTalkClient) UploadMedia(mediaType string, filename string, reader io.Reader) (media MediaResponse, err error) {
	upload := UploadFile{
		FieldName: "media",
		FileName:  filename,
		Reader:    reader,
	}
	params := url.Values{}
	params.Add("type", mediaType)
	c.HTTPClient.Timeout = 120 * time.Second
	err = c.httpRPC("media/upload", params, upload, &media)
	return media, err
}

//DownloadMedia is to download a media file from DingTalk
func (c *DingTalkClient) DownloadMedia(mediaID string, write io.Writer) error {
	var data MediaResponse
	data.Writer = write
	params := url.Values{}
	params.Add("media_id", mediaID)
	c.HTTPClient.Timeout = 120 * time.Second
	err := c.httpRPC("media/get", params, nil, &data)
	return err
}
