package cos

import (
	"context"
	"net/http"
	"time"
)

// Client 客户端， cos的句柄
type Client struct {
	conn *Conn
}

// New cos包的入口
func New(o *Option) *Client {
	client := Client{}
	conf := getDefaultConf()
	conf.AppID = o.AppID
	conf.SecretID = o.SecretID
	conf.SecretKey = o.SecretKey
	conf.Region = o.Region

	if o.Domain != "" {
		conf.Domain = o.Domain
	}

	conn := Conn{&http.Client{}, conf}
	client.conn = &conn

	return &client
}

// GetTimeoutCtx 获取一个带超时的context
func GetTimeoutCtx(timeout time.Duration) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), timeout)

	return ctx
}

// Bucket get bucket
func (c *Client) Bucket(name string) *Bucket {
	return &Bucket{name, c.conn}
}

// GetBucketList 获取bucketlist
func (c *Client) GetBucketList(ctx context.Context) (*ListAllMyBucketsResult, error) {
	req, err := http.NewRequest(http.MethodGet, "http://service.cos.myqcloud.com/", nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	c.conn.signHeader(req, nil, nil)
	res, err := c.conn.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	res, err = checkHTTPErr(res)
	if err != nil {
		return nil, err
	}

	labr := &ListAllMyBucketsResult{}
	err = XMLDecode(res.Body, labr)
	if err != nil {
		return nil, err
	}

	return labr, err
}

// CreateBucket 建立bucket
func (c *Client) CreateBucket(ctx context.Context, name string, acl *AccessControl) error {
	res, err := c.conn.Do(ctx, http.MethodPut, name, "", nil, acl.GenHead(), nil)
	if err == nil {
		defer res.Body.Close()
	}

	return err
}

// DeleteBucket delete a bucket
func (c *Client) DeleteBucket(ctx context.Context, name string) error {
	_, err := c.conn.Do(ctx, http.MethodDelete, name, "", nil, nil, nil)

	return err
}

// GetBucketACL get bucket's acl
func (c *Client) GetBucketACL(ctx context.Context, name string) (*AccessControlPolicy, error) {
	params := map[string]interface{}{"acl": ""}
	res, err := c.conn.Do(ctx, http.MethodGet, name, "", params, nil, nil)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	aclp := &AccessControlPolicy{}

	err = XMLDecode(res.Body, aclp)
	if err != nil {
		return nil, err
	}

	return aclp, nil
}

// SetBucketACL set bucket's acl
func (c *Client) SetBucketACL(ctx context.Context, name string, acl *AccessControl) error {
	params := map[string]interface{}{"acl": ""}
	res, err := c.conn.Do(ctx, http.MethodPut, name, "", params, acl.GenHead(), nil)
	if err == nil {
		defer res.Body.Close()
	}

	return err
}

// BucketExists bucket exists?
func (c *Client) BucketExists(ctx context.Context, name string) error {
	res, err := c.conn.Do(ctx, http.MethodHead, name, "", nil, nil, nil)
	if err == nil {
		defer res.Body.Close()
	}

	return err
}

// ListBucketContents list
func (c *Client) ListBucketContents(ctx context.Context, name string, qc *QueryCondition) (*ListBucketResult, error) {
	resp, err := c.conn.Do(ctx, http.MethodGet, name, "", qc.GenParams(), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	lbr := &ListBucketResult{}
	err = XMLDecode(resp.Body, lbr)
	if err != nil {
		return nil, err
	}

	return lbr, nil
}

// ListUploading list uploading task
func (c *Client) ListUploading(ctx context.Context, bucket string, lu *ListUploadParam) (*ListMultipartUploadsResult, error) {
	res, err := c.conn.Do(ctx, http.MethodGet, bucket, "", lu.GenParams(), nil, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	lmur := &ListMultipartUploadsResult{}
	err = XMLDecode(res.Body, lmur)
	if err != nil {
		return nil, err
	}

	return lmur, nil
}
