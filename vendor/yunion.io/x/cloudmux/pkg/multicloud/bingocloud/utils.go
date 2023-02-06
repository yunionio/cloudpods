package bingocloud

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type portRange struct {
	Start int64
	End   int64
}

type securityGroupInput struct {
	GroupId                    string `json:"GroupId,omitempty"`
	IpProtocol                 string `json:"IpProtocol,omitempty"`
	CidrIp                     string `json:"CidrIp,omitempty"`
	FromPort                   string `json:"FromPort,omitempty"`
	ToPort                     string `json:"ToPort,omitempty"`
	BoundType                  string `json:"BoundType,omitempty"`
	SourceSecurityGroupOwnerId string `json:"SourceSecurityGroupOwnerId,omitempty"`
	SourceSecurityGroupName    string `json:"SourceSecurityGroupName,omitempty"`
	Policy                     string `json:"Policy,omitempty"`
}

func securityRuleToBingoCloud(secGroupId string, rule cloudprovider.SecurityRule) []securityGroupInput {
	var portRanges []portRange
	if len(rule.Ports) == 0 {
		var start, end = 0, 0
		if rule.PortStart <= 0 {
			start = 1
		} else {
			start = rule.PortStart
		}

		if rule.PortEnd <= 0 {
			end = 65535
		} else {
			end = rule.PortEnd
		}

		portRanges = append(portRanges, portRange{int64(start), int64(end)})
	}

	for i := range rule.Ports {
		port := rule.Ports[i]
		if port <= 0 {
			portRanges = append(portRanges, portRange{1, 65535})
		} else {
			portRanges = append(portRanges, portRange{int64(port), int64(port)})
		}
	}

	protocol := ""
	if rule.Protocol == secrules.PROTO_ANY {
		protocol = "all"
	} else {
		protocol = rule.Protocol
	}

	var boundType = "In"
	var policy = "DROP"
	var allInputs []securityGroupInput

	if rule.Direction == secrules.SecurityRuleEgress {
		boundType = "Out"
	}
	if rule.Action == secrules.SecurityRuleAllow {
		policy = "ACCEPT"
	}

	for _, port := range portRanges {
		input := securityGroupInput{
			GroupId:    secGroupId,
			IpProtocol: protocol,
			CidrIp:     rule.IPNet.String(),
			FromPort:   strconv.FormatInt(port.Start, 10),
			ToPort:     strconv.FormatInt(port.End, 10),
			BoundType:  boundType,
			Policy:     policy,
		}
		allInputs = append(allInputs, input)
	}

	return allInputs
}

func nextDeviceName(curDeviceNames []string) (string, error) {
	var currents []string
	for _, item := range curDeviceNames {
		currents = append(currents, strings.ToLower(item))
	}

	for i := 0; i < 25; i++ {
		device := fmt.Sprintf("/dev/vd%c", byte(98+i))
		found := false
		for _, item := range currents {
			if strings.HasPrefix(item, device) {
				found = true
			}
		}

		if !found {
			return device, nil
		}
	}

	return "", fmt.Errorf("disk devicename out of index, current deivces: %s", currents)
}
