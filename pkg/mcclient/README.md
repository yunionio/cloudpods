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
		)

	params := jsonutils.NewDict()
	params.Set("scope", jsonutils.NewString("system"))

	result, err := modules.Servers.List(s, params)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", jsonutils.Marshal(result).PrettyString())
}
```

使用统一API入口调用

```golang
package main

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

// 文档说明: https://www.cloudpods.org/docs/development/apisdk/apigateway/
func main() {
	client := mcclient.NewClient("https://10.127.100.2/api/s/identity/v3", // 注意此地址不带端口
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
		"apigateway", // 注意此endpoint类型
		token,
		)


	params := jsonutils.NewDict()
	params.Set("scope", jsonutils.NewString("system"))

	result, err := modules.Servers.List(s, params)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", jsonutils.Marshal(result).PrettyString())
}
```
