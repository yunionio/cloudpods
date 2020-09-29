Yunion OneCloud API go library
===============================


Sample code

    :::golang
    package main

    import (
        "context"
        "fmt"

        "yunion.io/x/onecloud/pkg/mcclient"
        "yunion.io/x/onecloud/pkg/mcclient/modules"
    )

    func main() {
        client := mcclient.NewClient("https://<onecloud_controller_ip>:30500/v3",
            60,
            true,
            true,
            "",
            "")
        token, err := client.Authenticate("sysadmin", "<password>", "Default", "system", "Default")
        if err != nil {
            panic(err)
        }
        s := client.NewSession(context.Background(),
            "region0",
            "",
            "PublicURL",
            token,
            "")

        result, err := modules.Servers.List(s, nil)
        if err != nil {
            panic(err)
        }
        fmt.Printf("%#v\n", result)
    }


