package credentials

type AccessKeyCredential struct {
	AccessKeyId     string
	AccessKeySecret string
}

func NewAccessKeyCredential(accessKeyId, accessKeySecret string) *AccessKeyCredential {
	return &AccessKeyCredential{
		AccessKeyId:     accessKeyId,
		AccessKeySecret: accessKeySecret,
	}
}
