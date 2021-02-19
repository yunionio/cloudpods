package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/organizations"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

/*
 * {"arn":"arn:aws:organizations::285906155448:account/o-vgh74bqhdw/285906155448","email":"swordqiu@gmail.com","id":"285906155448","joined_method":"INVITED","joined_timestamp":"2021-02-09T03:55:27.724000Z","name":"qiu jian","status":"ACTIVE"}
 */
type SAccount struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Arn    string `json:"arn"`
	Email  string `json:"email"`
	Status string `json:"status"`

	JoinedMethod    string    `json:"joined_method"`
	JoinedTimestamp time.Time `json:"joined_timestamp"`

	IsMaster bool `json:"is_master"`
}

func (r *SRegion) ListAccounts() ([]SAccount, error) {
	orgCli, err := r.getOrganizationClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetOrganizationClient")
	}
	input := organizations.DescribeOrganizationInput{}
	orgOutput, err := orgCli.DescribeOrganization(&input)
	if err != nil {
		log.Errorf("%#v", err)
		return nil, errors.Wrap(err, "DescribeOrganization")
	}

	var nextToken *string
	accounts := make([]SAccount, 0)
	for {
		input := organizations.ListAccountsInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}
		parts, err := orgCli.ListAccounts(&input)
		if err != nil {
			return nil, errors.Wrap(err, "ListAccounts")
		}
		for _, actPtr := range parts.Accounts {
			account := SAccount{
				ID:              *actPtr.Id,
				Name:            *actPtr.Name,
				Arn:             *actPtr.Arn,
				Email:           *actPtr.Email,
				Status:          *actPtr.Status,
				JoinedMethod:    *actPtr.JoinedMethod,
				JoinedTimestamp: *actPtr.JoinedTimestamp,
			}
			if *orgOutput.Organization.MasterAccountId == *actPtr.Id {
				account.IsMaster = true
			}
			accounts = append(accounts, account)
		}
		if parts.NextToken == nil || len(*parts.NextToken) == 0 {
			break
		} else {
			nextToken = parts.NextToken
		}
	}
	return accounts, nil
}

func (self *SAwsClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	defRegion := self.getDefaultRegion()
	if defRegion == nil {
		return nil, errors.Wrap(errors.ErrInvalidStatus, "no valid default region")
	}
	accounts, err := defRegion.ListAccounts()
	if err != nil {
		// find errors
		if strings.Contains(err.Error(), "AWSOrganizationsNotInUseException") || strings.Contains(err.Error(), "AccessDeniedException") {
			// permission denied, fall back to single account mode
			subAccount := cloudprovider.SSubAccount{}
			subAccount.Name = self.cpcfg.Name
			subAccount.Account = self.accessKey
			subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
			return []cloudprovider.SSubAccount{subAccount}, nil
		} else {
			return nil, errors.Wrap(err, "ListAccounts")
		}
	} else {
		// check if caller is a root caller
		caller, _ := self.GetCallerIdentity()
		isRootAccount := false
		// arn:aws:iam::285906155448:root
		if caller != nil && strings.HasSuffix(caller.Arn, ":root") {
			log.Debugf("root %s", caller.Arn)
			isRootAccount = true
		}
		subAccounts := []cloudprovider.SSubAccount{}
		for _, account := range accounts {
			subAccount := cloudprovider.SSubAccount{}
			if account.Status == "ACTIVE" {
				subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
			} else {
				subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_SUSPENDED
			}
			if account.IsMaster {
				subAccount.Name = self.cpcfg.Name
				subAccount.Account = self.accessKey
			} else {
				if isRootAccount {
					log.Warningf("Cannot access non-master account with root account!!")
					subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NO_PERMISSION
				}
				subAccount.Name = fmt.Sprintf("%s/%s", account.Name, account.ID)
				subAccount.Account = fmt.Sprintf("%s/%s", self.accessKey, account.ID)
			}
			subAccounts = append(subAccounts, subAccount)
		}
		return subAccounts, nil
	}
}
