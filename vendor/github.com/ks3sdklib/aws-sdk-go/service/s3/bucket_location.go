package s3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/aws/awserr"
	"io"
	"regexp"
)

var reBucketLocation = regexp.MustCompile(`>([^<>]+)<\/LocationConstraint`)

func buildGetBucketLocation(r *aws.Request) {
	if r.DataFilled() {
		out := r.Data.(*GetBucketLocationOutput)
		b, err := io.ReadAll(r.HTTPResponse.Body)
		if err != nil {
			r.Error = awserr.New("Unmarshal",
				"failed reading response body", err)
			return
		}
		match := reBucketLocation.FindSubmatch(b)
		if len(match) > 1 {
			loc := string(match[1])
			out.LocationConstraint = aws.String(loc)
		}
	}
}
