// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package qcloud // import "yunion.io/x/cloudmux/pkg/multicloud/qcloud"

/*
Network Pre Conditions

若是不开放公网，则需要开放以下域名的443和80端口, 避免部分资源同步异常或创建资源失败

cvm.tencentcloudapi.com
cvm.ap-shanghai-fsi.tencentcloudapi.com
cvm.ap-shenzhen-fsi.tencentcloudapi.com

vpc.tencentcloudapi.com
vpc.ap-shanghai-fsi.tencentcloudapi.com
vpc.ap-shenzhen-fsi.tencentcloudapi.com

cloudaudit.tencentcloudapi.com
cloudaudit.ap-shanghai-fsi.tencentcloudapi.com
cloudaudit.ap-shenzhen-fsi.tencentcloudapi.com

cbs.tencentcloudapi.com
cbs.ap-shanghai-fsi.tencentcloudapi.com
cbs.ap-shenzhen-fsi.tencentcloudapi.com

account.api.qcloud.com

clb.tencentcloudapi.com
clb.ap-shanghai-fsi.tencentcloudapi.com
clb.ap-shenzhen-fsi.tencentcloudapi.com

cdb.tencentcloudapi.com
cdb.ap-shanghai-fsi.tencentcloudapi.com
cdb.ap-shenzhen-fsi.tencentcloudapi.com

mariadb.tencentcloudapi.com
mariadb.ap-shanghai-fsi.tencentcloudapi.com
mariadb.ap-shenzhen-fsi.tencentcloudapi.com

postgres.tencentcloudapi.com
postgres.ap-shanghai-fsi.tencentcloudapi.com
postgres.ap-shenzhen-fsi.tencentcloudapi.com

sqlserver.tencentcloudapi.com
sqlserver.ap-shanghai-fsi.tencentcloudapi.com
sqlserver.ap-shenzhen-fsi.tencentcloudapi.com

lb.api.qcloud.com
wss.api.qcloud.com
cns.api.qcloud.com
vpc.api.qcloud.com
billing.tencentcloudapi.com
cam.tencentcloudapi.com
monitor.tencentcloudapi.com

cos.ap-beijing.myqcloud.com
service.cos.myqcloud.com
cos.ap-bangkok.myqcloud.com.
bj.file.myqcloud.com.
tj.file.myqcloud.com.
cd.file.myqcloud.com.
cq.file.myqcloud.com.
gz.file.myqcloud.com.
hk.file.myqcloud.com.
cos.ap-mumbai.myqcloud.com.
cos.ap-nanjing.myqcloud.com.
cos.ap-seoul.myqcloud.com.
sh.file.myqcloud.com.
cos.ap-shanghai-fsi.myqcloud.com.
cos.ap-shenzhen-fsi.myqcloud.com.
sgp.file.myqcloud.com.
cos.ap-tokyo.myqcloud.com.
ger.file.myqcloud.com.
cos.eu-moscow.myqcloud.com.
cos.na-ashburn.myqcloud.com.
cos.na-siliconvalley.myqcloud.com.
ca.file.myqcloud.com.
3jytr9q1.dayugslb.com.
60icvusw.dayugslb.com.
5ekm872f.dayugslb.com.
r5mfqoyl.dayugslb.com.

Permission Pre Conditions
若仅需要同步资源, 则需要赋予账号以下权限
QCloudFinanceFullAccess
QcloudCVMReadOnlyAccess
QcloudCamReadOnlyAccess
QcloudCOSReadOnlyAccess
QcloudAuditReadOnlyAccess
QcloudMonitorReadOnlyAccess

若需要资源读写，需要赋予账号以下权限
QCloudFinanceFullAccess
QcloudCVMFullAccess
QcloudCamFullAccess
QcloudCBSFullAccess
QcloudEIPFullAccess
QcloudCOSFullAccess
QcloudAuditFullAccess
QcloudMonitorFullAccess
*/
