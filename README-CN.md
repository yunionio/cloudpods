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

Cloudpods提供了如下的功能：

* 管理多云资源的功能，可以管理大多数的主流云，包括私有云，例如OpenStack，以及公有云，例如AWS，Azure，GCP，阿里云，华为云和腾讯云等
* 允许以统一的联邦身份访问各个云平台的原生控制台的SSO
* 一个可以管理海量KVM虚拟机的轻量级私有云
* 一个能进行物理机全生命周期管理的裸机云
* 实现了VMware vSphere虚拟化集群的自助服务和自动化
* 一套功能丰富、统一一致的RESTAPI和模型访问以上的云资源和功能
* 一套完整的多租户认证和访问控制体系
* 自动将镜像转换为不同云平台需要的格式的多云镜像服务

### 支持的云平台

* 公有云:
  * AWS
  * Azure
  * Google Cloud Platform
  * 阿里云
  * 华为云
  * 腾讯云
  * UCloud
  * 中国电信云
  * 中国移动云
  * 京东云
* 私有云:
  * OpenStack
  * ZStack
  * AlibabaCloud Aspara (阿里飞天)
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

我们可以通过以下简单两步将Cloudpods安装在一台至少8GiB内存和100GB硬盘的Linux主机上（目前CentOS 7和Debian 10经过充分测试）

(下面假设该主机的IP为 *10.168.26.216*)

### 1. 准备SSH免密登录

```bash
# Generate the local ssh keypair
# (SKIP this stekp if you already have ~/.ssh/id_rsa.pub locally)
$ ssh-keygen
# Copy the generated ~/.ssh/id_rsa.pub public key to the machine to be deployed
$ ssh-copy-id -i ~/.ssh/id_rsa.pub root@10.168.26.216
# Try to login to the machine to be deployed without password,
# should be able to get the hostname of the deployed machine
# without entering the login password
$ ssh root@10.168.26.216 "hostname"
```

### 2. 安装Cloudpods

通过以下命令开始安装Cloudpods：

```bash
# Git clone the ocboot installation tool locally
$ git clone https://github.com/yunionio/ocboot && cd ./ocboot && ./run.py 10.168.26.216
```

大概10-30分钟后，安装完成。访问 https://10.168.26.216 登入Cloudpods的Web控制台。初始的账号为 *admin* ，密码为 *admin@123*

请参考文档 [快速开始](https://www.cloudpods.org/zh/docs/quickstart/) 获得更详细的安装指导。

## 文档

* [Cloudpods文档](https://www.cloudpods.org/zh)

* [Swagger API文档](https://www.cloudpods.org/zh/docs/swagger/)

## 谁在使用Cloudpods？

请在[这里](https://github.com/yunionio/cloudpods/issues/11427)查看Cloudpods用户列表。如果你正在使用Cloudpods，欢迎回复留下你的信息。谢谢对Cloudpods的支持！

## 联系我们

您可以通过如下方式联系我们：

* Reddit: [r/Cloudpods](https://www.reddit.com/r/Cloudpods/)

* 哔哩哔哩: [Cloudpods](https://space.bilibili.com/623431553/)

* 微信: 请扫描如下二维码联系我们

<img src="https://www.cloudpods.org/images/skillcode.png" alt="WeChat QRCode">

## 版本历史

请访问[Cloudpods Changelog](https://www.cloudpods.org/zh/docs/changelog/).

## 开发规划

请访问[Cloudpods Roadmap](https://www.cloudpods.org/zh/docs/roadmap/).

## 贡献

欢迎和感谢任何形式的贡献，不局限于贡献代码，流程细节请查看 [CONTRIBUTING](./CONTRIBUTING_zh.md)。

## License

Apache license 2.0，详情请看 [LICENSE](./LICENSE)。
