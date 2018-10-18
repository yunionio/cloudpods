package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/pkg/util/secrules"
)

func AppendFilter(filters []*ec2.Filter, name string, values []string) ([]*ec2.Filter) {
	f := &ec2.Filter{}
	v := make([]*string, len(values))
	for _, value := range values {
		v = append(v, &value)
	}

	f.SetName(name)
	f.SetValues(v)
	return append(filters, f)
}

func AppendSingleValueFilter(filters []*ec2.Filter, name string, value string) ([]*ec2.Filter) {
	f := &ec2.Filter{}
	f.SetName(name)
	f.SetValues([]*string{&value})
	return append(filters, f)
}

func ConvertedList(list []string) ([]*string) {
	result := make([]*string, len(list))
	for _, item := range list {
		result = append(result, &item)
	}

	return result
}

func ConvertedPointList(list []*string) ([]string) {
	result := make([]string, len(list))
	for _, item := range list {
		if item != nil {
			result = append(result, *item)
		}
	}

	return result
}

// Security Rule Transform
func AwsIpPermissionToYunion(permission ec2.IpPermission) secrules.SecurityRule {
	return secrules.SecurityRule{}
}

func YunionIpPermissionToAws(rule secrules.SecurityRule) ec2.IpPermission {
	return ec2.IpPermission{}
}

func DiffIpPermission(permission ec2.IpPermission, rule secrules.SecurityRule) {
	
}