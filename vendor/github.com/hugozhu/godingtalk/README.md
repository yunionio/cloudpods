# DingTalk Open API golang SDK

![image](http://static.dingtalk.com/media/lALOAQ6nfSvM5Q_229_43.png)

Check out DingTalk Open API document at: https://ding-doc.dingtalk.com/

## Usage

Fetch the SDK
```
export GOPATH=`pwd`
go get github.com/hugozhu/godingtalk
```

### Example code to send a micro app message

```
package main

import (
    "github.com/hugozhu/godingtalk"
    "log"
    "os"
)

func main() {
    c := godingtalk.NewDingTalkClient(os.Getenv("corpid"), os.Getenv("corpsecret"))
    c.RefreshAccessToken()
    err := c.SendAppMessage(os.Args[1], os.Args[2], os.Args[3])
    if err != nil {
        log.Println(err)
    }
}
```


## Guide

Step-by-step Guide to use this SDK

http://hugozhu.myalert.info/2016/05/02/66-use-dingtalk-golang-sdk-to-send-message-on-pi.html

## Tools

**ding_alert** : Command line tool to send app/text/oa ... messages

```
export GOPATH=`pwd`
go get github.com/hugozhu/godingtalk/demo/ding_alert

export corpid=<组织的corpid 通过 https://oa.dingtalk.com 获取>
export corpsecret=<组织的corpsecret 通过 https://oa.dingtalk.com 获取>

./bin/ding_alert
Usage of ./bin/ding_alert:
  -agent string
    	agent Id (default "22194403")
  -chat string
    	chat id (default "chat6a93bc1ee3b7d660d372b1b877a9de62")
  -file string
    	file path for media message
  -link string
    	link url (default "http://hugozhu.myalert.info/dingtalk")
  -sender string
    	sender id (default "011217462940")
  -text string
    	text for link message (default "This is link text")
  -title string
    	title for link message (default "This is link title")
  -touser string
    	touser id (default "0420506555")
  -type string
    	message type (app, text, image, voice, link, oa) (default "app")

```

**github**: Deliver Github webhook events to DingTalk, which can be deployed on Google AppEngine.

more info at: http://hugozhu.myalert.info/2016/05/15/67-use-free-google-cloud-service-to-deliver-github-webhook-events-to-dingtalk.html

```
export GOPATH=`pwd`
go get github.com/hugozhu/godingtalk/demo/github/appengine
```

Modify `app.yaml`

```
cd src/github.com/hugozhu/godingtalk/demo/github/appengine
cat app.yaml
application: github-alert-<random_number>
version: 1
runtime: go
api_version: go1
env_variables:
  CORP_ID: '<从 http://oa.dingtalk.com 获取>'
  CORP_SECRET: '<从 http://oa.dingtalk.com 获取>'
  GITHUB_WEBHOOK_SECRET: '<从 http://github.com/ 获取>'
  SENDER_ID: '<从 http://open.dingtalk.com 调用api获取>'
  CHAT_ID: '<从 http://open.dingtalk.com 调用api获取>'
handlers:
- url: /.*
  script: _go_app

```



