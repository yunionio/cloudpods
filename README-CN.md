# Cloudpods

[![CircleCI](https://circleci.com/gh/yunionio/cloudpods.svg?style=svg)](https://circleci.com/gh/yunionio/cloudpods)
[![Build Status](https://travis-ci.com/yunionio/cloudpods.svg?branch=master)](https://travis-ci.org/yunionio/cloudpods)
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/cloudpods)](https://goreportcard.com/report/github.com/yunionio/cloudpods)

## Cloudpods是什么?

<img src="https://www.cloudpods.org/images/cloudpods_logo_green.png" alt="Cloudpods" height="100">

Cloudpods是一个开源的Golang实现的云原生的融合多云/混合云的云平台，也就是一个“云上之云”。Cloudpods不仅可以管理本地的虚拟机和物理机资源，还可以管理多个云平台和云账号。Cloudpods隐藏了这些异构基础设施资源的数据模型和API的差异，对外暴露了一套统一的API，允许用户就像用一个云一样地访问多云。从而大大降低了访问多云的复杂度，提升了管理多云的效率。

## 谁需要Cloudpods?

* 将几台物理服务器虚拟化成一个私有云平台
* 需要一个紧凑而且功能相对完整的物理机全生命周期管理工具
* 将VMware vSphere虚拟化集群转换为一个可以自服务的私有云平台
* 在混合云的场景，能够在一个界面访问私有云和公有云
* 通过一个集中的入口访问分布在多个公有云平台上的多个账号
* 当前只使用一个云公有云账号但希望将来使用多云的用户

## 功能

### 概览 & UI 展示

<details>
  <summary>管理多云资源的功能，可以管理大多数的主流云，包括私有云，例如OpenStack，以及公有云，例如AWS，Azure，GCP，阿里云，华为云和腾讯云等</summary>
  <ul>
    <li>
      <p>云帐号纳管</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/cloudselect.png" alt="multi cloud management">
    </li>
    <li>
      <p>云帐号列表</p>
      <img src="https://i.imgur.com/Q0LipfI.png" alt="cloud account list">
    </li>
    <li>
      <p>公有云虚拟机列表</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/publicvmlist.png">
    </li>
  </ul>
</details>

<details>
  <summary>
  一个可以管理海量KVM虚拟机的轻量级私有云
  </summary>
  <ul>
    <li>
      <p>虚拟机列表</p>
      <img src="https://i.imgur.com/DbkRUoo.png">
    </li>
    <li>
      <p>虚拟机创建页面</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/createkvmvm1.png">
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/createkvmvm2.png">
    </li>
    <li>
      <p>虚拟机可通过 VNC 或者 SSH 登录</p>
      <img src="https://i.imgur.com/m0rkeQ3.png">
    </li>
    <li>
      <p>宿主机列表</p>
      <img src="https://imgur.com/i509t5a.png">
    </li>
    <li>
      <p>镜像模板列表</p>
      <img src="https://imgur.com/UVFLGi2.png">
    </li>
    <li>
      <p>VPC列表</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/vpclist.png">
    </li>
    <li>
      <p>二层网络列表（经典网络）</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/wirelist.png">
    </li>
    <li>
      <p>IP子网列表</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/ipsubnetlist.png">
    </li> 
    <li>
      <p>弹性公网IP列表（VPC网络）</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/vpclist.png">
    </li>
    <li>
      <p>LB列表</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/lblist.png">
    </li>
  </ul>
</details>

<details>
  <summary>
  一个能进行物理机全生命周期管理的裸机云
  </summary>
  <ul>
    <li>
      <p>物理机列表</p>
      <img src="https://i.imgur.com/Jz8b5nC.png">
    </li>
    <li>
      <p>物理机纳管</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/addphysicalmachine.png">
    </li>
    <li>
      <p>安装操作系统</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/createbaremetal.png">
    </li>
    <li>支持 ARM64 的物理机服务器</li>
  </ul>
</details>

<details>
  <summary>一套完整的多租户认证和访问控制体系</summary>
  <ul>
    <li>
      <p>支持本地sql、LDAP 等认证源</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/idplist.png">
    </li>
    <li>
      <p>多租户系统，包括域，项目，组，用户，角色和权限等</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/domainlist.png">
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/projectlist.png">
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/grouplist.png">
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/userlist.png">
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/rolelist.png">
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/policylist.png">
    </li>
  </ul>
</details>

<details>
  <summary>
  实现了VMware vSphere虚拟化集群的自助服务和自动化
  </summary>
  <ul>
    <li>
      <p>添加VMware云账号</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/createvmware.png">
    </li>
    <li>
      <p>VMware云账号列表</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/vmwarelist.png">
    </li>
    <li>
      <p>自动创建二层网络</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/vmwarewirelist.png">
    </li>
    <li>
      <p>自动创建IP子网</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/vmwareipsubnetlist.png">
    </li>
    <li>
      <p>新建VMware虚拟机</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/createvmwarevm1.png">
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/createvmwarevm2.png">
    </li>
  </ul>
</details>


<details>
  <summary>
  允许以统一的联邦身份访问各个云平台的原生控制台的SSO
  </summary>
  <ul>
    <li>
      <p>为云账号开启免密登录（以阿里云为例）</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/enablecloudsso.png">
    </li>
    <li>
      <p>将Cloudpods平台用户添加为免密登录用户</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/addsamluser.png">
    </li>
    <li>
      <p>多云统一登录入口</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/cloudssoentry.png">
    </li>
    <li>
      <p>多云统一登录-免密登录用户列表</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/cloudsamluser.png">
    </li>
    <li>
      <p>Cloudpods平台用户免密登录阿里云</p>
    </li>
  </ul>
</details>

<details>
  <summary>
  一套功能丰富、统一一致的RESTAPI和模型访问以上的云资源和功能
  </summary>
</details>

<details>
  <summary>
  自动将镜像转换为不同云平台需要的格式的多云镜像服务
  </summary>
</details>


### 支持的云平台

* 公有云:
  * AWS
  * Azure
  * Google Cloud Platform
  * 阿里云
  * 华为云
  * 腾讯云
  * UCloud
  * 天翼云
  * 移动云
  * 京东云
* 私有云:
  * OpenStack
  * ZStack
  * Alibaba Cloud Aspara (阿里飞天)
  * Huawei HCSO (华为HCSO)
  * Nutanix
* 本地基础设施资源:
  * KVM
  * VMWare vSphere vCenter/ESXi
  * Baremetals (IPMI, Redfish API)
  * Object storages (Minio, Ceph, XSky)
  * NAS (Ceph)

### 支持的云资源

* Servers: instances, disks, network interfaces, networks, vpcs, storages, hosts, wires, snapshots, snapshot policies, security groups, elastic IPs, SSH keypairs, images
* Load Balancers: instances, listeners, backend groups, backends, TSL certificates, ACLs
* Object Storage: buckets, objects
* NAS: file_systems, access_groups, mount_targets
* RDS: instances, accounts, backups, databases, parameters, privileges
* Elastic Cache: instances, accounts, backups, parameters
* DNS: DNS zones, DNS records
* VPC: VPCs, VPC peering, inter-VPC network, NAT gateway, DNAT/SNAT rules, route tables, route entries

## 快速开始

我们可以通过以下简单三步将Cloudpods安装在一台至少8GiB内存和100GB硬盘的Linux主机上（目前CentOS 7和Debian 10经过充分测试）

(下面假设该主机的IP为 *10.168.26.216*)

### 1. 准备SSH免密登录

```bash
# 生成ssh密钥对
# (如果已经有~/.ssh/id_rsa和~/.ssh/id_rsa.pub,请跳过此步。请确保ssh密钥私钥未设置密码)
$ ssh-keygen -t rsa -N ''
# 将生成的ssh公钥~/.ssh/id_rsa.pub拷贝到待部署的目标主机
$ ssh-copy-id -i ~/.ssh/id_rsa.pub root@10.168.26.216
# 测试SSH免密登录是否生效。免密登录到待部署主机执行hostname命令
# 如果设置成功，执行下面命令能够回显待部署主机的主机名
# 不需要输入待部署主机的密码
$ ssh root@10.168.26.216 "hostname"
```

### 2. 安装git和相关工具

#### CentOS 7安装git和epel源
```bash
yum install -y git epel-release ansible
```

#### Debian 10安装git
```bash
apt install -y git ansible
```

### 3. 安装Cloudpods

通过以下命令开始安装Cloudpods：

```bash
# Git clone the ocboot installation tool locally
$ git clone -b release/3.8 https://github.com/yunionio/ocboot && cd ./ocboot && ./run.py 10.168.26.216
```

大概10-30分钟后，安装完成。访问 https://10.168.26.216 登入Cloudpods的Web控制台。初始的账号为 *admin* ，密码为 *admin@123*

请参考文档 [快速开始](https://www.cloudpods.org/zh/docs/quickstart/allinone/) 获得更详细的安装指导。

已经有 Kubernetes 集群？可以尝试 [使用 Helm 安装 Cloudpods](https://www.cloudpods.org/zh/docs/quickstart/k8s/)。

## 文档

* [Cloudpods文档](https://www.cloudpods.org/zh)

* [Swagger API文档](https://www.cloudpods.org/zh/docs/swagger/)

## 谁在使用Cloudpods？

请在[这里](https://github.com/yunionio/cloudpods/issues/11427)查看Cloudpods用户列表。如果你正在使用Cloudpods，欢迎回复留下你的信息。谢谢对Cloudpods的支持！

## 联系我们

您可以通过如下方式联系我们：

* [服务订阅](https://www.yunion.cn/subscription/index.html)

* 哔哩哔哩: [Cloudpods](https://space.bilibili.com/623431553/)

* 微信: 请扫描如下二维码联系我们

<img src="https://www.cloudpods.org/images/contact_me_qr_20210701.png" alt="WeChat QRCode">

## 版本历史

请访问[Cloudpods Changelog](https://www.cloudpods.org/zh/docs/changelog/).

## 开发规划

请访问[Cloudpods Roadmap](https://www.cloudpods.org/zh/docs/roadmap/).

## 贡献

欢迎和感谢任何形式的贡献，不局限于贡献代码，流程细节请查看 [CONTRIBUTING](./CONTRIBUTING_zh.md)。

## License

Apache license 2.0，详情请看 [LICENSE](./LICENSE)。
