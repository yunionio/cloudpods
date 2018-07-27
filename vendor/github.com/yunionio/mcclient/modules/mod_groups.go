package modules

import (
	"fmt"

	"github.com/yunionio/mcclient"
)

type GroupManager struct {
	ResourceManager
}

func (this *GroupManager) GetUsers(s *mcclient.ClientSession, gid string) (*ListResult, error) {
	url := fmt.Sprintf("/groups/%s/users", gid)
	return this._list(s, url, "users")
}

var (
	Groups GroupManager
)

func init() {
	Groups = GroupManager{NewIdentityV3Manager("group", "groups",
		[]string{},
		[]string{"ID", "Name", "Domain_Id", "Description"})}

	register(&Groups)
}
