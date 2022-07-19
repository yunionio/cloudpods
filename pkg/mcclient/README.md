Yunion OneCloud API go library
===============================


Login to first controlplane node of your cluster and execute `ocadm cluster rcadmin` to get auth info.

For example:

```bash
$ ocadm cluster rcadmin
export OS_AUTH_URL=https://10.127.100.2:30500/v3
export OS_USERNAME=sysadmin
export OS_PASSWORD=7AQMP9H2umQvbxxx
export OS_PROJECT_DOMAIN=default
export OS_PROJECT_NAME=system
export YUNION_INSECURE=true
export OS_REGION_NAME=region0
export OS_ENDPOINT_TYPE=publicURL
```

Sample code

```golang 
package main

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

func main() {
	client := mcclient.NewClient("https://10.127.100.2:30500/v3",
		60,
		true,
		true,
		"",
		"")
	token, err := client.Authenticate("sysadmin", "7AQMP9H2umQvbxxx", "Default", "system", "Default")
	if err != nil {
		panic(err)
	}
	s := client.NewSession(context.Background(),
		"region0",
		"",
		"publicURL",
		token,
		"")

	result, err := modules.Servers.List(s, nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", jsonutils.Marshal(result).PrettyString())
}
```
