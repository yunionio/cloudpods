package cos

// AccessControl privilige
type AccessControl struct {
	ACL         string
	GrantRead   string
	GrantWrite  string
	FullControl string
}

// GenHead 生成http head
func (acl *AccessControl) GenHead() map[string]string {
	header := map[string]string{
		"x-cos-acl":                acl.ACL,
		"x-cos-grant-read":         acl.GrantRead,
		"x-cos-grant-write":        acl.GrantWrite,
		"x-cos-grant-full-control": acl.FullControl,
	}

	for k, v := range header {
		if v == "" {
			delete(header, k)
		}
	}

	return header
}

// QueryCondition query condition
type QueryCondition struct {
	Prefix       string
	Delimiter    string
	EncodingType string
	Marker       string
	MaxKeys      int
}

// GenParams generate params:map[string]interface{}
func (qc *QueryCondition) GenParams() map[string]interface{} {
	params := map[string]interface{}{
		"prefix":        qc.Prefix,
		"delimiter":     qc.Delimiter,
		"encoding-type": qc.EncodingType,
		"marker":        qc.Marker,
		"max-keys":      qc.MaxKeys,
	}

	for k, v := range params {
		if v == "" {
			delete(params, k)
		}
		if v == 0 {
			delete(params, k)
		}
	}

	return params
}

// ListUploadParam list upload param
type ListUploadParam struct {
	Prefix         string
	Delimiter      string
	EncodingType   string
	MaxUploads     int
	KeyMarker      string
	UploadIDMarker string
}

// GenParams generate params for request
func (lup *ListUploadParam) GenParams() map[string]interface{} {
	params := map[string]interface{}{
		"prefix":           lup.Prefix,
		"delimiter":        lup.Delimiter,
		"encoding-type":    lup.EncodingType,
		"max-uploads":      lup.MaxUploads,
		"key-marker":       lup.KeyMarker,
		"upload-id-marker": lup.UploadIDMarker,
	}

	for k, v := range params {
		if v == "" {
			delete(params, k)
		}

		if v == 0 {
			delete(params, k)
		}
	}
	params["uploads"] = ""

	return params

}

// CompleteMultipartUpload compelete slice upload
type CompleteMultipartUpload struct {
	Part []struct {
		PartNumber int
		ETag       string
	}
}
