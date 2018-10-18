package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/pkg/util/secrules"
	"net"
	"fmt"
	"yunion.io/x/log"
	"strings"
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

func isAwsPermissionAllPorts(p ec2.IpPermission) bool {
	//  全部端口范围： TCP/UDP （0，65535）    其他：（-1，-1）
	if (*p.IpProtocol == "tcp" || *p.IpProtocol == "udp") && *p.FromPort == 0 && *p.ToPort == 65535 {
		return true
	} else if *p.FromPort == -1 && *p.ToPort == -1 {
		return true
	} else {
		return false
	}
}

// Security Rule Transform
func AwsIpPermissionToYunion(direction secrules.TSecurityRuleDirection,p ec2.IpPermission) ([]secrules.SecurityRule, error) {

	if len(p.UserIdGroupPairs) > 0 {
		return nil, fmt.Errorf("AwsIpPermissionToYunion not supported aws rule: UserIdGroupPairs specified")
	}

	if len(p.PrefixListIds) > 0 {
		return nil, fmt.Errorf("AwsIpPermissionToYunion not supported aws rule: PrefixListIds specified")
	}

	if len(p.Ipv6Ranges) > 0 {
		log.Debugf("AwsIpPermissionToYunion ignored IPV6 rule: %s", p.Ipv6Ranges)
	}

	rules := []secrules.SecurityRule{}
	isAllPorts := isAwsPermissionAllPorts(p)
	for _, ip := range p.IpRanges {
		ipNet := strings.Split(*ip.CidrIp, "/")
		if len(ipNet) != 2 {
			log.Debugf("AwsIpPermissionToYunion ignored IPV4 rule: %s", *ip.CidrIp)
			continue
		}

		var rule secrules.SecurityRule
		if isAllPorts {
			rule = secrules.SecurityRule{
				Action:      secrules.SecurityRuleAllow,
				IPNet:       &net.IPNet{net.IP(ipNet[0]), net.IPMask(ipNet[1])},
				Protocol:    *p.IpProtocol,
				Direction:   direction,
				Description: "",
			}
		} else {
			rule = secrules.SecurityRule{
				Action:      secrules.SecurityRuleAllow,
				IPNet:       &net.IPNet{net.IP(ipNet[0]), net.IPMask(ipNet[1])},
				Protocol:    *p.IpProtocol,
				Direction:   direction,
				PortStart:   int(*p.FromPort),
				PortEnd:     int(*p.ToPort),
				Description: "",
			}
		}

		rules = append(rules, rule)

	}

	return rules, nil
}

func YunionSecRuleToAws(rule secrules.SecurityRule) ec2.IpPermission {
	return ec2.IpPermission{}
}

func DiffIpPermission(permission ec2.IpPermission, rule secrules.SecurityRule) {

}