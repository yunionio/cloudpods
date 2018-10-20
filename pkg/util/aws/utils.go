package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/pkg/util/secrules"
	"net"
	"fmt"
	"yunion.io/x/log"
	"strings"
)

type portRange struct {
	Start int64
	End   int64
}

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
	result := make([]*string, 0)
	for _, item := range list {
		if len(item) > 0 {
			result = append(result, &item)
		}
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

func StrVal(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}

func IntVal(s *int64) int64 {
	if s != nil {
		return *s
	}

	return 0
}

func isAwsPermissionAllPorts(p ec2.IpPermission) bool {
	if p.FromPort == nil || p.ToPort == nil {
		return false
	}

	//  全部端口范围： TCP/UDP （0，65535）    其他：（-1，-1）
	if (*p.IpProtocol == "tcp" || *p.IpProtocol == "udp") && *p.FromPort == 0 && *p.ToPort == 65535 {
		return true
	} else if *p.FromPort == -1 && *p.ToPort == -1 {
		return true
	} else {
		return false
	}
}

func awsProtocolToYunion(p ec2.IpPermission) string {
	if p.IpProtocol != nil && *p.IpProtocol == "-1" {
		return secrules.PROTO_ANY
	} else {
		return *p.IpProtocol
	}
}

func isYunionRuleAllPorts(r secrules.SecurityRule) bool {
	//  全部端口范围： TCP/UDP （0，65535）    其他：（-1，-1）
	if (r.Protocol == "tcp" || r.Protocol == "udp") && r.PortStart == 0 && r.PortEnd == 65535 {
		return true
	} else if r.PortStart == -1 && r.PortEnd == -1 {
		return true
	} else {
		return false
	}
}

func yunionPortRangeToAws(r secrules.SecurityRule) ([]portRange) {
	// port 0 / -1 都代表所有端口
	portranges := []portRange{}
	if len(r.Ports) == 0 {
		var start, end = 0, 0
		if r.PortStart <= 0 {
			if r.Protocol == "tcp" || r.Protocol == "udp" {
				start = 0
			} else  {
				start = -1
			}
		} else {
			start = r.PortStart
		}

		if r.PortEnd <= 0 {
			if r.Protocol == "tcp" || r.Protocol == "udp" {
				end = 65535
			} else  {
				end = -1
			}
		} else {
			end = r.PortEnd
		}

		portranges = append(portranges, portRange{int64(start), int64(end)})
	}

	for _, port := range r.Ports {
		if port <= 0 && ( r.Protocol == "tcp" || r.Protocol == "udp" ) {
			portranges = append(portranges, portRange{0, 65535})
		} else if port <= 0 {
			portranges = append(portranges, portRange{-1, -1})
		} else {
			portranges = append(portranges, portRange{int64(port), int64(port)})
		}
	}

	return portranges
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
	protocol := awsProtocolToYunion(p)
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
				Protocol:    protocol,
				Direction:   direction,
				Priority:    1,
				Description: StrVal(ip.Description),
			}
		} else {
			rule = secrules.SecurityRule{
				Action:      secrules.SecurityRuleAllow,
				IPNet:       &net.IPNet{net.IP(ipNet[0]), net.IPMask(ipNet[1])},
				Protocol:    protocol,
				Direction:   direction,
				Priority:    1,
				Description: StrVal(ip.Description),
			}

			if p.FromPort != nil {
				rule.PortStart = int(*p.FromPort)
			}

			if p.ToPort != nil {
				rule.PortStart = int(*p.ToPort)
			}
		}

		rules = append(rules, rule)

	}

	return rules, nil
}

func YunionSecRuleToAws(rule secrules.SecurityRule) ([]ec2.IpPermission, error) {
	if rule.Action == secrules.SecurityRuleDeny {
		return nil, fmt.Errorf("YunionSecRuleToAws ignored  aws not supported deny rule")
	}

	iprange := rule.IPNet.String()
	if iprange ==  "<nil>" {
		return nil, fmt.Errorf("YunionSecRuleToAws ignored  ipnet should not be empty")
	}
	ipranges := []*ec2.IpRange{}
	ipranges = append(ipranges, &ec2.IpRange{CidrIp: &iprange, Description: &rule.Description})

	portranges := yunionPortRangeToAws(rule)
	permissions := []ec2.IpPermission{}
	for _, port := range portranges {
		permission := ec2.IpPermission{
			FromPort:         &port.Start,
			IpProtocol:       &rule.Protocol,
			IpRanges:         ipranges,
			ToPort:           &port.End,
		}

		permissions = append(permissions, permission)
	}

	return permissions, nil
}

func awsTagSpecification(resourceType string,) {

}