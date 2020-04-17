package ovn

import (
	"fmt"
)

func vpcLrName(vpcId string) string {
	return fmt.Sprintf("vpc-r-%s", vpcId)
}

func vpcHostLsName(vpcId string) string {
	return fmt.Sprintf("vpc-h-%s", vpcId)
}

func vpcRhpName(vpcId string) string {
	return fmt.Sprintf("vpc-rh-%s", vpcId)
}

func vpcHrpName(vpcId string) string {
	return fmt.Sprintf("vpc-hr-%s", vpcId)
}
