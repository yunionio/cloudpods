package s3

import (
	"fmt"
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/internal/apierr"
	"hash"
	"strconv"
)

func CheckUploadCrc64(r *aws.Request) {
	clientCrc := r.Crc64.Sum64()
	serverCrc := uint64(0)
	if r.HTTPResponse.Header.Get("X-Amz-Checksum-Crc64ecma") != "" {
		serverCrc, _ = strconv.ParseUint(r.HTTPResponse.Header.Get("X-Amz-Checksum-Crc64ecma"), 10, 64)
	}

	r.Config.LogInfo("client crc:%d, server crc:%d", clientCrc, serverCrc)

	if serverCrc != 0 && clientCrc != serverCrc {
		r.Error = apierr.New("CRCCheckError", fmt.Sprintf("client crc and server crc do not match, request id:[%s]", r.HTTPResponse.Header.Get("X-Kss-Request-Id")), nil)
		r.Config.LogError("%s", r.Error.Error())
	}
}

func CheckDownloadCrc64(s3 *S3, res *GetObjectOutput, crc hash.Hash64) error {
	var err error
	clientCrc := crc.Sum64()
	serverCrc := uint64(0)
	if res.Metadata["X-Amz-Checksum-Crc64ecma"] != nil {
		serverCrc, _ = strconv.ParseUint(*res.Metadata["X-Amz-Checksum-Crc64ecma"], 10, 64)
	}

	s3.Config.LogInfo("client crc:%d, server crc:%d", clientCrc, serverCrc)

	if serverCrc != 0 && clientCrc != serverCrc {
		err = apierr.New("CRCCheckError", fmt.Sprintf("client crc and server crc do not match, request id:[%s]", *res.Metadata["X-Kss-Request-Id"]), nil)
		s3.Config.LogError("%s", err.Error())
	}

	return err
}
